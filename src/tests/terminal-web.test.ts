import { describe, expect, test } from "bun:test"
import { fixtureStoryIds } from "../data/fixtures.ts"
import { MAX_CONTROL_BYTES, parseTerminalControl } from "../dev/terminal-web/protocol.ts"
import {
  buildStorySpawn,
  consumeInboundBudget,
  createSession,
  createTerminalWebServer,
  disposeSessionChild,
  INBOUND_BURST_BYTES,
  INBOUND_REFILL_BYTES_PER_SECOND,
  type ManagedChild,
  type ManagedTerminal,
  type TerminalWebInstrumentation,
  type TerminalWebRuntime,
} from "../dev/terminal-web/server.ts"
import { MAX_INPUT_BYTES } from "../dev/terminal-web/protocol.ts"

const stripAnsi = (value: string) => value
  .replace(/\u001B(?:\][^\u0007]*(?:\u0007|\u001B\\)|\[[0-?]*[ -/]*[@-~]|[@-_])/g, "")
  .replace(/\r/g, "")

const waitForCondition = async (predicate: () => boolean, timeoutMs = 3_000) => {
  const deadline = Date.now() + timeoutMs
  while (!predicate()) {
    if (Date.now() >= deadline) throw new Error("Timed out waiting for condition")
    await Bun.sleep(15)
  }
}

class SocketProbe {
  raw = ""
  statuses: Array<{ status: string; detail?: string }> = []
  private listeners = new Set<() => void>()

  note(data: unknown) {
    if (typeof data === "string") {
      try {
        const parsed = JSON.parse(data) as { type?: string; status?: string; detail?: string }
        if (parsed.type === "status" && parsed.status) this.statuses.push({ status: parsed.status, detail: parsed.detail })
      } catch {
        // Only server status packets are text; malformed packets are irrelevant to terminal output.
      }
    } else if (data instanceof ArrayBuffer) {
      this.raw += new TextDecoder().decode(new Uint8Array(data), { stream: true })
    } else if (ArrayBuffer.isView(data)) {
      this.raw += new TextDecoder().decode(new Uint8Array(data.buffer, data.byteOffset, data.byteLength), { stream: true })
    }
    for (const listener of this.listeners) listener()
  }

  async waitFor(predicate: () => boolean, timeoutMs = 5_000) {
    if (predicate()) return
    await new Promise<void>((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.listeners.delete(check)
        reject(new Error("Timed out waiting for WebSocket output"))
      }, timeoutMs)
      const check = () => {
        if (!predicate()) return
        clearTimeout(timeout)
        this.listeners.delete(check)
        resolve()
      }
      this.listeners.add(check)
    })
  }
}

const connect = async (origin: string, token: string) => {
  const wsOrigin = origin.replace(/^http/, "ws")
  const socket = new WebSocket(`${wsOrigin}/pty?token=${encodeURIComponent(token)}`, { headers: { Origin: origin } } as never)
  socket.binaryType = "arraybuffer"
  const probe = new SocketProbe()
  socket.addEventListener("message", (event) => probe.note(event.data))
  await new Promise<void>((resolve, reject) => {
    socket.addEventListener("open", () => resolve(), { once: true })
    socket.addEventListener("error", () => reject(new Error("WebSocket failed to open")), { once: true })
  })
  return { socket, probe }
}

const closeSocket = async (socket: WebSocket) => {
  if (socket.readyState === WebSocket.CLOSED) return
  await new Promise<void>((resolve) => {
    socket.addEventListener("close", () => resolve(), { once: true })
    socket.close(1000, "test complete")
  })
}

const waitForSocketClose = async (socket: WebSocket, timeoutMs = 3_000) => {
  if (socket.readyState === WebSocket.CLOSED) return { code: 1006, reason: "already closed" }
  return await new Promise<{ code: number; reason: string }>((resolve, reject) => {
    const timeout = setTimeout(() => reject(new Error("Timed out waiting for WebSocket close")), timeoutMs)
    socket.addEventListener("close", (event) => {
      clearTimeout(timeout)
      resolve({ code: event.code, reason: event.reason })
    }, { once: true })
  })
}

const getToken = async (harness: Awaited<ReturnType<typeof createTerminalWebServer>>) => {
  const html = await (await fetch(harness.url)).text()
  const token = html.match(/name="stacks-terminal-token" content="([^"]+)"/)?.[1]
  if (!token) throw new Error("Harness page did not contain its token")
  return token
}

type LifecycleCounts = {
  terminalsCreated: number
  terminalsClosed: number
  childrenSpawned: number
  childrenDisposed: number
}

const lifecycleInstrumentation = (counts: LifecycleCounts): TerminalWebInstrumentation => ({
  onTerminalCreated: () => { counts.terminalsCreated += 1 },
  onTerminalClosed: () => { counts.terminalsClosed += 1 },
  onChildSpawned: () => { counts.childrenSpawned += 1 },
  onChildDisposed: () => { counts.childrenDisposed += 1 },
})

const makeFakeTerminal = (onClose?: () => void): ManagedTerminal => {
  const terminal = {
    closed: false,
    close: () => {
      terminal.closed = true
      onClose?.()
    },
    resize: () => undefined,
    write: () => 0,
    setRawMode: () => undefined,
  }
  return terminal as ManagedTerminal
}

const makeKillableChild = (signals: Array<number | NodeJS.Signals>): ManagedChild => {
  let resolveExit: (code: number) => void = () => undefined
  const child = {
    exitCode: null as number | null,
    exited: new Promise<number>((resolve) => { resolveExit = resolve }),
    kill: (signal?: number | NodeJS.Signals) => {
      signals.push(signal ?? "SIGTERM")
      child.exitCode = 0
      resolveExit(0)
    },
  }
  return child as ManagedChild
}

describe("terminal web protocol", () => {
  test("accepts only exact, bounded control messages", () => {
    expect(parseTerminalControl('{"type":"start","storyId":"mixed","cols":80,"rows":24}')).toEqual({ type: "start", storyId: "mixed", cols: 80, rows: 24 })
    expect(parseTerminalControl('{"type":"resize","cols":40,"rows":20}')).toEqual({ type: "resize", cols: 40, rows: 20 })
    expect(parseTerminalControl('{"type":"restart","storyId":"large-stack","cols":160,"rows":40}')).toEqual({ type: "restart", storyId: "large-stack", cols: 160, rows: 40 })
    expect(parseTerminalControl('{"type":"start","storyId":"../../anything","cols":80,"rows":24}')).toBeNull()
    expect(parseTerminalControl('{"type":"resize","cols":1,"rows":24}')).toBeNull()
    expect(parseTerminalControl('{"type":"resize","cols":501,"rows":24}')).toBeNull()
    expect(parseTerminalControl('{"type":"resize","cols":80,"rows":301}')).toBeNull()
    expect(parseTerminalControl('{"type":"resize","cols":80,"rows":24,"command":"sh"}')).toBeNull()
    expect(parseTerminalControl("x".repeat(MAX_CONTROL_BYTES + 1))).toBeNull()
  })

  test("story launch is fixed to allowlisted fixtures, argv and a tiny explicit env", () => {
    expect(fixtureStoryIds).toHaveLength(11)
    const spawn = buildStorySpawn("mixed")
    expect(spawn.cmd).toEqual([process.execPath, "--preload", "@opentui/solid/preload", expect.stringMatching(/src\/stories\/index\.tsx$/)])
    expect(spawn.cwd).toMatch(/stacks$/)
    expect(spawn.env).toEqual({ TERM: "xterm-256color", COLORTERM: "truecolor", STACKS_STORY: "mixed" })
    expect(() => buildStorySpawn("not-a-story")).toThrow("allowlist")
  })

  test("child cleanup is idempotent and only signals the direct child", async () => {
    const session = createSession()
    let signals = 0
    let closes = 0
    const child = {
      exitCode: null as number | null,
      exited: Promise.resolve(0),
      kill: (_signal?: number | NodeJS.Signals) => {
        signals += 1
        child.exitCode = 0
      },
    }
    const terminal = {
      closed: false,
      close: () => {
        closes += 1
        terminal.closed = true
      },
      resize: () => undefined,
      write: () => 0,
      setRawMode: () => undefined,
    }
    session.child = child as never
    session.terminal = terminal as never

    await Promise.all([disposeSessionChild(session), disposeSessionChild(session), disposeSessionChild(session)])
    expect(signals).toBe(1)
    expect(closes).toBe(1)
    expect(session.child).toBeUndefined()
    expect(session.terminal).toBeUndefined()
  })

  test("rate limits sustained aggregate input, then refills deterministically", () => {
    const session = createSession(0)
    expect(consumeInboundBudget(session, INBOUND_BURST_BYTES, 0)).toBe(true)
    expect(consumeInboundBudget(session, 1, 0)).toBe(false)
    expect(consumeInboundBudget(session, INBOUND_REFILL_BYTES_PER_SECOND, 1_000)).toBe(true)
    expect(consumeInboundBudget(session, 1, 1_000)).toBe(false)
  })
})

describe("terminal web server", () => {
  test("rejects unauthenticated origins, malformed controls and oversized frames", async () => {
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false })
    let malformed: Awaited<ReturnType<typeof connect>> | undefined
    let oversized: Awaited<ReturnType<typeof connect>> | undefined
    try {
      const token = await getToken(harness)
      const wrongToken = await fetch(new URL("/pty?token=wrong", harness.url), { headers: { Origin: harness.origin } })
      const wrongOrigin = await fetch(new URL(`/pty?token=${token}`, harness.url), { headers: { Origin: "http://127.0.0.1:1" } })
      const missingOrigin = await fetch(new URL(`/pty?token=${token}`, harness.url))
      expect(wrongToken.status).toBe(403)
      expect(wrongOrigin.status).toBe(403)
      expect(missingOrigin.status).toBe(403)

      malformed = await connect(harness.origin, token)
      const malformedClose = waitForSocketClose(malformed.socket)
      malformed.socket.send('{"type":"resize","cols":80,"rows":24,"extra":true}')
      expect((await malformedClose).code).toBe(1008)
      malformed = undefined

      oversized = await connect(harness.origin, token)
      const oversizedClose = waitForSocketClose(oversized.socket)
      oversized.socket.send(new Uint8Array(MAX_INPUT_BYTES + 1))
      // Bun's native maxPayloadLength transport currently tears this down with
      // 1006 before an app close frame can be written; 1009 is also valid when
      // the manual guard gets the frame first.
      expect([1006, 1009]).toContain((await oversizedClose).code)
      oversized = undefined
    } finally {
      if (malformed) await closeSocket(malformed.socket)
      if (oversized) await closeSocket(oversized.socket)
      await harness.close()
    }
  }, 10_000)

  test("closes a socket that sustains aggregate control input beyond its token bucket", async () => {
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false })
    let peer: Awaited<ReturnType<typeof connect>> | undefined
    try {
      peer = await connect(harness.origin, await getToken(harness))
      const close = waitForSocketClose(peer.socket, 8_000)
      const resize = JSON.stringify({ type: "resize", cols: 80, rows: 24 })
      const burstCount = Math.ceil((INBOUND_BURST_BYTES * 2) / Buffer.byteLength(resize))
      for (let index = 0; index < burstCount; index += 1) peer.socket.send(resize)
      expect((await close).code).toBe(1008)
      peer = undefined
    } finally {
      if (peer) await closeSocket(peer.socket)
      await harness.close()
    }
  }, 12_000)

  test("spawn failure closes the created PTY and reports error without marking the session started", async () => {
    const counts: LifecycleCounts = { terminalsCreated: 0, terminalsClosed: 0, childrenSpawned: 0, childrenDisposed: 0 }
    const runtime: TerminalWebRuntime = {
      createTerminal: () => makeFakeTerminal(),
      spawnStory: () => { throw new Error("forced spawn failure") },
    }
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false, runtime, instrumentation: lifecycleInstrumentation(counts) })
    let peer: Awaited<ReturnType<typeof connect>> | undefined
    try {
      peer = await connect(harness.origin, await getToken(harness))
      peer.socket.send(JSON.stringify({ type: "start", storyId: "mixed", cols: 80, rows: 24 }))
      await peer.probe.waitFor(() => peer!.probe.statuses.some(({ status, detail }) => status === "error" && detail === "forced spawn failure"))
      expect(peer.probe.statuses.some(({ status }) => status === "started")).toBe(false)
      expect(counts).toEqual({ terminalsCreated: 1, terminalsClosed: 1, childrenSpawned: 0, childrenDisposed: 0 })
      await closeSocket(peer.socket)
      await waitForCondition(() => harness.activeSessionCount === 0 && harness.activeChildCount === 0)
      peer = undefined
    } finally {
      if (peer) await closeSocket(peer.socket)
      await harness.close()
    }
  }, 10_000)

  test("serializes a burst of real restarts, resets before each replacement, and disposes every resource", async () => {
    const counts: LifecycleCounts = { terminalsCreated: 0, terminalsClosed: 0, childrenSpawned: 0, childrenDisposed: 0 }
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false, instrumentation: lifecycleInstrumentation(counts) })
    let peer: Awaited<ReturnType<typeof connect>> | undefined
    try {
      peer = await connect(harness.origin, await getToken(harness))
      peer.socket.send(JSON.stringify({ type: "start", storyId: "mixed", cols: 80, rows: 24 }))
      await peer.probe.waitFor(() => stripAnsi(peer!.probe.raw).includes("STACKS UI LAB"))

      for (const storyId of ["all-ready", "queued", "large-stack", "mixed"]) {
        peer.socket.send(JSON.stringify({ type: "restart", storyId, cols: 80, rows: 24 }))
      }
      await peer.probe.waitFor(() => peer!.probe.statuses.filter(({ status }) => status === "restarted").length === 4, 12_000)
      const statuses = peer.probe.statuses.map(({ status }) => status)
      expect(statuses.filter((status) => status === "reset")).toHaveLength(4)
      let resetsSeen = 0
      for (const status of statuses) {
        if (status === "reset") resetsSeen += 1
        if (status === "restarted") expect(resetsSeen).toBeGreaterThan(0)
      }
      expect(counts.terminalsCreated).toBe(5)
      expect(counts.childrenSpawned).toBe(5)

      await closeSocket(peer.socket)
      await waitForCondition(() => harness.activeSessionCount === 0 && harness.activeChildCount === 0)
      expect(counts.terminalsClosed).toBe(5)
      expect(counts.childrenDisposed).toBe(5)
      peer = undefined
    } finally {
      if (peer) await closeSocket(peer.socket)
      await harness.close()
    }
  }, 20_000)

  test("burst restart plus immediate disconnect/server close awaits direct-child cleanup", async () => {
    const counts: LifecycleCounts = { terminalsCreated: 0, terminalsClosed: 0, childrenSpawned: 0, childrenDisposed: 0 }
    const signals: Array<number | NodeJS.Signals> = []
    const runtime: TerminalWebRuntime = {
      createTerminal: () => makeFakeTerminal(),
      spawnStory: () => makeKillableChild(signals),
    }
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false, runtime, instrumentation: lifecycleInstrumentation(counts) })
    const peer = await connect(harness.origin, await getToken(harness))
    peer.socket.send(JSON.stringify({ type: "start", storyId: "mixed", cols: 80, rows: 24 }))
    await peer.probe.waitFor(() => peer.probe.statuses.some(({ status }) => status === "started"))
    for (const storyId of ["all-ready", "queued", "large-stack", "mixed"]) {
      peer.socket.send(JSON.stringify({ type: "restart", storyId, cols: 80, rows: 24 }))
    }

    // This is the same close path SIGTERM uses: mark closed, await the queue,
    // then direct-signal the final child before the server resolves shutdown.
    peer.socket.close(1000, "burst disconnect")
    await harness.close()
    expect(harness.activeSessionCount).toBe(0)
    expect(harness.activeChildCount).toBe(0)
    expect(counts.terminalsCreated).toBe(counts.terminalsClosed)
    expect(counts.childrenSpawned).toBe(counts.childrenDisposed)
    expect(signals).toEqual(["SIGTERM"])
  }, 10_000)

  test("closing sessions remain observable until a SIGTERM'd direct child actually exits", async () => {
    const signals: Array<number | NodeJS.Signals> = []
    let releaseExit: (() => void) | undefined
    const runtime: TerminalWebRuntime = {
      createTerminal: () => makeFakeTerminal(),
      spawnStory: () => {
        const child = {
          exitCode: null as number | null,
          exited: new Promise<number>((resolve) => { releaseExit = () => { child.exitCode = 0; resolve(0) } }),
          kill: (signal?: number | NodeJS.Signals) => { signals.push(signal ?? "SIGTERM") },
        }
        return child as ManagedChild
      },
    }
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false, runtime })
    const token = await getToken(harness)
    const peer = await connect(harness.origin, token)
    peer.socket.send(JSON.stringify({ type: "start", storyId: "mixed", cols: 80, rows: 24 }))
    await peer.probe.waitFor(() => peer.probe.statuses.some(({ status }) => status === "started"))

    const shutdown = harness.close()
    await waitForCondition(() => signals.length === 1)
    expect(harness.activeSessionCount).toBe(1)
    expect(harness.activeChildCount).toBe(1)
    const lateUpgrade = await fetch(new URL(`/pty?token=${token}`, harness.url), { headers: { Origin: harness.origin } })
    expect(lateUpgrade.status).toBe(503)
    releaseExit?.()
    await shutdown
    expect(harness.activeSessionCount).toBe(0)
    expect(harness.activeChildCount).toBe(0)
    expect(signals).toEqual(["SIGTERM"])
  }, 10_000)

  test("serves the browser client and drives the real OpenTUI fixture process over a binary PTY", async () => {
    const harness = await createTerminalWebServer({ port: 0, installSignalHandlers: false })
    let first: Awaited<ReturnType<typeof connect>> | undefined
    let second: Awaited<ReturnType<typeof connect>> | undefined
    try {
      const page = await fetch(harness.url)
      const html = await page.text()
      expect(page.status).toBe(200)
      expect(html).toContain('id="terminal"')
      expect(html).toContain("80×24")
      const token = html.match(/name="stacks-terminal-token" content="([^"]+)"/)?.[1]
      expect(token).toBeTruthy()

      const [client, xtermCss, harnessCss] = await Promise.all([
        fetch(new URL("/client.js", harness.url)),
        fetch(new URL("/xterm.css", harness.url)),
        fetch(new URL("/terminal.css", harness.url)),
      ])
      expect(client.status).toBe(200)
      expect((await client.text()).length).toBeGreaterThan(100_000)
      expect(await xtermCss.text()).toContain(".xterm")
      expect(await harnessCss.text()).toContain("oklch")

      first = await connect(harness.origin, token!)
      first.socket.send(JSON.stringify({ type: "start", storyId: "mixed", cols: 80, rows: 24 }))
      await first.probe.waitFor(() => stripAnsi(first!.probe.raw).includes("STACKS UI LAB"))
      expect(first.probe.raw).toContain("\u001b")

      first.socket.send(JSON.stringify({ type: "resize", cols: 40, rows: 20 }))
      first.socket.send(Uint8Array.of(0x1b, 0x5b, 0x43))
      await first.probe.waitFor(() => stripAnsi(first!.probe.raw).includes("#185 Keep service identity explicit"))

      first.socket.send(Uint8Array.of(0x71))
      await first.probe.waitFor(() => first!.probe.statuses.some(({ status }) => status === "exited"))
      expect(first.probe.statuses.some(({ status, detail }) => status === "exited" && detail === "code 0")).toBe(true)
      await closeSocket(first.socket)
      await waitForCondition(() => harness.activeSessionCount === 0 && harness.activeChildCount === 0)
      first = undefined

      second = await connect(harness.origin, token!)
      second.socket.send(JSON.stringify({ type: "start", storyId: "mixed", cols: 80, rows: 24 }))
      await second.probe.waitFor(() => stripAnsi(second!.probe.raw).includes("STACKS UI LAB"))
      await closeSocket(second.socket)
      await waitForCondition(() => harness.activeSessionCount === 0 && harness.activeChildCount === 0)
      second = undefined
    } finally {
      if (first) await closeSocket(first.socket)
      if (second) await closeSocket(second.socket)
      await harness.close()
    }
  }, 20_000)
})
