import { timingSafeEqual } from "node:crypto"
import { resolve } from "node:path"
import { fixtureStories, isFixtureStoryId } from "../../data/fixtures.ts"
import { MAX_INPUT_BYTES, parseTerminalControl, type TerminalControl, type TerminalSize } from "./protocol.ts"

const HOSTNAME = "127.0.0.1"
const DEV_DIRECTORY = import.meta.dir
const PROJECT_ROOT = resolve(DEV_DIRECTORY, "../../..")
const STORY_ENTRY = resolve(PROJECT_ROOT, "src/stories/index.tsx")
const CLIENT_ENTRY = resolve(DEV_DIRECTORY, "client.ts")
const XTERM_CSS = resolve(PROJECT_ROOT, "node_modules/@xterm/xterm/css/xterm.css")
const HARNESS_CSS = resolve(DEV_DIRECTORY, "terminal.css")
const MAX_WS_PAYLOAD = MAX_INPUT_BYTES
export const INBOUND_BURST_BYTES = 16 * 1024
export const INBOUND_REFILL_BYTES_PER_SECOND = 8 * 1024

export type ManagedTerminal = Pick<Bun.Terminal, "closed" | "close" | "resize" | "write" | "setRawMode">
export type ManagedChild = Pick<Bun.Subprocess, "exitCode" | "exited" | "kill">

export type StorySpawn = {
  cmd: string[]
  cwd: string
  env: Record<string, string>
}

export type TerminalWebRuntime = {
  createTerminal(options: Bun.TerminalOptions): ManagedTerminal
  spawnStory(spawn: StorySpawn, terminal: ManagedTerminal): ManagedChild
}

export type TerminalWebInstrumentation = {
  onTerminalCreated?(): void
  onTerminalClosed?(): void
  onChildSpawned?(): void
  onChildDisposed?(): void
}

export type Session = {
  started: boolean
  closed: boolean
  child?: ManagedChild
  terminal?: ManagedTerminal
  disposing?: Promise<void>
  controlQueue: Promise<void>
  closePromise?: Promise<void>
  generation: number
  inboundTokens: number
  inboundLastRefillAt: number
  pendingChildDisposals: number
}

type SocketSession = Session & {
  socket?: Bun.ServerWebSocket<SocketSession>
}

export type TerminalWebServer = {
  readonly url: URL
  readonly origin: string
  readonly activeSessionCount: number
  readonly activeChildCount: number
  close(): Promise<void>
}

export type CreateTerminalWebServerOptions = {
  port?: number
  installSignalHandlers?: boolean
  /** Test-only adapters let lifecycle failure paths be exercised without a real child. */
  runtime?: TerminalWebRuntime
  instrumentation?: TerminalWebInstrumentation
}

function createToken() {
  return Buffer.from(crypto.getRandomValues(new Uint8Array(32))).toString("base64url")
}

function tokensMatch(actual: string | null, expected: string) {
  if (!actual) return false
  const actualBytes = Buffer.from(actual)
  const expectedBytes = Buffer.from(expected)
  return actualBytes.byteLength === expectedBytes.byteLength && timingSafeEqual(actualBytes, expectedBytes)
}

function isSameOrigin(request: Request) {
  const origin = request.headers.get("origin")
  if (!origin) return false
  try {
    const requestUrl = new URL(request.url)
    const originUrl = new URL(origin)
    return requestUrl.protocol === originUrl.protocol
      && requestUrl.hostname === HOSTNAME
      && originUrl.hostname === HOSTNAME
      && requestUrl.port === originUrl.port
  } catch {
    return false
  }
}

export function buildStorySpawn(storyId: string): StorySpawn {
  if (!isFixtureStoryId(storyId)) throw new Error("Refusing to launch a story outside the fixture allowlist.")
  return {
    cmd: [process.execPath, "--preload", "@opentui/solid/preload", STORY_ENTRY],
    cwd: PROJECT_ROOT,
    env: {
      TERM: "xterm-256color",
      COLORTERM: "truecolor",
      STACKS_STORY: storyId,
    },
  }
}

export function createSession(now = Date.now()): Session {
  return {
    started: false,
    closed: false,
    controlQueue: Promise.resolve(),
    generation: 0,
    inboundTokens: INBOUND_BURST_BYTES,
    inboundLastRefillAt: now,
    pendingChildDisposals: 0,
  }
}

/** A small per-socket token bucket prevents sustained input/queue abuse. */
export function consumeInboundBudget(session: Session, byteLength: number, now = Date.now()) {
  const elapsedMs = Math.max(0, now - session.inboundLastRefillAt)
  const refill = (elapsedMs / 1_000) * INBOUND_REFILL_BYTES_PER_SECOND
  session.inboundTokens = Math.min(INBOUND_BURST_BYTES, session.inboundTokens + refill)
  session.inboundLastRefillAt = now
  if (!Number.isFinite(byteLength) || byteLength < 0 || byteLength > session.inboundTokens) return false
  session.inboundTokens -= byteLength
  return true
}

async function waitForChildExit(child: ManagedChild) {
  const result = await Promise.race([
    child.exited.then(() => "exited" as const),
    Bun.sleep(700).then(() => "timeout" as const),
  ])
  if (result === "timeout" && child.exitCode === null) {
    try {
      child.kill("SIGKILL")
    } catch {
      // The direct child may have won the race and exited between the checks.
    }
    await child.exited.catch(() => undefined)
  }
}

async function disposeResources(child: ManagedChild | undefined, terminal: ManagedTerminal | undefined, instrumentation?: TerminalWebInstrumentation) {
  if (child?.exitCode === null) {
    try {
      child.kill("SIGTERM")
    } catch {
      // Direct child already exited; its terminal still needs closing below.
    }
  }
  if (terminal && !terminal.closed) {
    try {
      terminal.close()
      instrumentation?.onTerminalClosed?.()
    } catch {
      // Closing an already-closing PTY is harmless in this dev-only bridge.
    }
  }
  if (child) {
    await waitForChildExit(child)
    instrumentation?.onChildDisposed?.()
  }
}

/** Safe to call more than once or concurrently; it never signals a process group. */
export async function disposeSessionChild(session: Session, instrumentation?: TerminalWebInstrumentation): Promise<void> {
  if (session.disposing) return session.disposing

  const child = session.child
  const terminal = session.terminal
  session.child = undefined
  session.terminal = undefined
  if (child) session.pendingChildDisposals += 1

  const disposal = (async () => {
    try {
      await disposeResources(child, terminal, instrumentation)
    } finally {
      if (child) session.pendingChildDisposals -= 1
    }
  })()

  session.disposing = disposal
  try {
    await disposal
  } finally {
    if (session.disposing === disposal) session.disposing = undefined
  }
}

function status(socket: Bun.ServerWebSocket<SocketSession> | undefined, state: "connected" | "started" | "reset" | "restarted" | "exited" | "error", detail?: string) {
  if (!socket) return
  socket.sendText(JSON.stringify({ type: "status", status: state, ...(detail ? { detail } : {}) }))
}

function terminalHtml(token: string) {
  const storyOptions = fixtureStories
    .map((story) => `<option value="${story.id}">${story.label}</option>`)
    .join("")
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="stacks-terminal-token" content="${token}">
    <title>Stacks · terminal lab</title>
    <link rel="stylesheet" href="/xterm.css">
    <link rel="stylesheet" href="/terminal.css">
  </head>
  <body>
    <main class="web-lab">
      <header class="dev-chrome" aria-label="Terminal lab controls">
        <span class="dev-label">stacks / terminal lab</span>
        <div class="preset-group" aria-label="Terminal dimensions">
          <button type="button" data-preset="40x20" aria-pressed="false">40×20</button>
          <button type="button" data-preset="80x24" aria-pressed="true">80×24</button>
          <button type="button" data-preset="120x30" aria-pressed="false">120×30</button>
          <button type="button" data-preset="160x40" aria-pressed="false">160×40</button>
          <button id="fit-terminal" type="button" aria-pressed="false">Fit</button>
        </div>
        <select id="story-select" aria-label="Fixture story">${storyOptions}</select>
        <button id="restart" type="button">Restart</button>
        <span class="spacer"></span>
        <span id="connection-status" class="connection-status" data-state="connecting" aria-live="polite">connecting</span>
        <button id="focus-terminal" type="button">Focus terminal</button>
      </header>
      <section class="terminal-stage" aria-label="Native OpenTUI terminal">
        <div id="terminal"></div>
      </section>
    </main>
    <script type="module" src="/client.js"></script>
  </body>
</html>`
}

async function clientBundle() {
  const build = await Bun.build({
    entrypoints: [CLIENT_ENTRY],
    target: "browser",
    format: "esm",
    minify: false,
    sourcemap: "none",
    env: "disable",
  })
  const artifact = build.outputs.find((output) => output.path.endsWith("client.js"))
  if (!build.success || !artifact) throw new Error("Could not build the terminal web client.")
  return artifact
}

function sizeDetail(size: TerminalSize, storyId?: string) {
  return `${storyId ? `${storyId} · ` : ""}${size.cols}×${size.rows}`
}

const nativeRuntime: TerminalWebRuntime = {
  createTerminal: (options) => new Bun.Terminal(options),
  spawnStory: (spawn, terminal) => Bun.spawn(spawn.cmd, {
    cwd: spawn.cwd,
    env: spawn.env,
    terminal: terminal as Bun.Terminal,
  }),
}

/**
 * Serialize every stateful operation for a socket. The stored tail always
 * settles, while the returned promise still lets the caller report failures.
 */
export function enqueueSessionControl(session: Session, operation: () => Promise<void>): Promise<void> {
  const task = session.controlQueue.then(async () => {
    if (session.closed) return
    await operation()
  })
  session.controlQueue = task.catch(() => undefined)
  return task
}

export async function createTerminalWebServer(options: CreateTerminalWebServerOptions = {}): Promise<TerminalWebServer> {
  const token = createToken()
  const client = await clientBundle()
  const runtime = options.runtime ?? nativeRuntime
  const instrumentation = options.instrumentation
  const sessions = new Set<SocketSession>()
  let closing = false
  let server: Bun.Server<SocketSession>

  const closeSession = async (session: SocketSession) => {
    if (session.closePromise) return session.closePromise
    session.closed = true
    const closingSession = (async () => {
      // A control already in flight may have attached a direct child. Wait for
      // it, then clean its exact resource pair before removing observability.
      await session.controlQueue
      await disposeSessionChild(session, instrumentation)
      sessions.delete(session)
    })()
    session.closePromise = closingSession
    return closingSession
  }

  const drainSessions = async () => {
    const failures: unknown[] = []
    while (sessions.size > 0) {
      const batch = await Promise.allSettled([...sessions].map(closeSession))
      failures.push(...batch
        .filter((result): result is PromiseRejectedResult => result.status === "rejected")
        .map((result) => result.reason))
      if (failures.length > 0) break
    }
    return failures
  }

  const reportFailure = (session: SocketSession, error: unknown) => {
    if (session.closed) return
    const detail = error instanceof Error ? error.message : "terminal operation failed"
    status(session.socket, "error", detail)
  }

  const monitorChildExit = (session: SocketSession, child: ManagedChild, terminal: ManagedTerminal, generation: number) => {
    void child.exited.then((code) => enqueueSessionControl(session, async () => {
      if (session.child !== child || session.generation !== generation) return
      session.child = undefined
      session.terminal = undefined
      await disposeResources(child, terminal, instrumentation)
      if (!session.closed) status(session.socket, "exited", `code ${code}`)
    })).catch((error) => reportFailure(session, error))
  }

  const spawnStory = async (session: SocketSession, control: Extract<TerminalControl, { storyId: string }>): Promise<boolean> => {
    const socket = session.socket
    if (!socket || session.closed) return false
    const spawn = buildStorySpawn(control.storyId)
    let terminal: ManagedTerminal | undefined
    let child: ManagedChild | undefined
    try {
      terminal = runtime.createTerminal({
        cols: control.cols,
        rows: control.rows,
        name: "xterm-256color",
        data: (_term, data) => {
          if (!session.closed && session.terminal === terminal) socket.sendBinary(data)
        },
      })
      instrumentation?.onTerminalCreated?.()
      terminal.setRawMode(true)
      if (session.closed) {
        await disposeResources(undefined, terminal, instrumentation)
        return false
      }
      child = runtime.spawnStory(spawn, terminal)
      instrumentation?.onChildSpawned?.()
      if (session.closed) {
        await disposeResources(child, terminal, instrumentation)
        return false
      }
      session.terminal = terminal
      session.child = child
      session.generation += 1
      monitorChildExit(session, child, terminal, session.generation)
      return true
    } catch (error) {
      await disposeResources(child, terminal, instrumentation)
      throw error
    }
  }

  const handleControl = async (session: SocketSession, control: TerminalControl) => {
    if (session.closed) return
    if (control.type === "start") {
      if (session.started) return
      const spawned = await spawnStory(session, control)
      if (!spawned || session.closed) return
      session.started = true
      status(session.socket, "started", sizeDetail(control, control.storyId))
      return
    }
    if (!session.started) return
    if (control.type === "resize") {
      session.terminal?.resize(control.cols, control.rows)
      return
    }
    // This frame is ordered before output from the replacement PTY. The
    // browser resets xterm when it sees it, so stale alt-screen content dies.
    status(session.socket, "reset", "restarting")
    await disposeSessionChild(session, instrumentation)
    if (session.closed) return
    const spawned = await spawnStory(session, control)
    if (!spawned || session.closed) return
    status(session.socket, "restarted", sizeDetail(control, control.storyId))
  }

  server = Bun.serve<SocketSession>({
    hostname: HOSTNAME,
    port: options.port ?? 3003,
    fetch(request, listener) {
      const url = new URL(request.url)
      if (url.pathname === "/pty") {
        if (closing) return new Response("Server shutting down", { status: 503 })
        if (!isSameOrigin(request) || !tokensMatch(url.searchParams.get("token"), token)) return new Response("Forbidden", { status: 403 })
        // Register before upgrading. A shutdown can then drain this session
        // even when open/close callbacks race the HTTP upgrade boundary.
        const session: SocketSession = { ...createSession() }
        sessions.add(session)
        const upgraded = listener.upgrade(request, { data: session })
        if (!upgraded) sessions.delete(session)
        return upgraded ? undefined : new Response("Upgrade failed", { status: 400 })
      }
      if (request.method !== "GET") return new Response("Method not allowed", { status: 405 })
      if (url.pathname === "/") {
        return new Response(terminalHtml(token), {
          headers: {
            "content-type": "text/html; charset=utf-8",
            "cache-control": "no-store",
            "content-security-policy": "default-src 'self'; connect-src 'self' ws://127.0.0.1:*; style-src 'self'; script-src 'self'; base-uri 'none'; frame-ancestors 'none'",
          },
        })
      }
      if (url.pathname === "/client.js") return new Response(client, { headers: { "content-type": "text/javascript; charset=utf-8", "cache-control": "no-store" } })
      if (url.pathname === "/xterm.css") return new Response(Bun.file(XTERM_CSS), { headers: { "content-type": "text/css; charset=utf-8", "cache-control": "no-store" } })
      if (url.pathname === "/terminal.css") return new Response(Bun.file(HARNESS_CSS), { headers: { "content-type": "text/css; charset=utf-8", "cache-control": "no-store" } })
      return new Response("Not found", { status: 404 })
    },
    websocket: {
      data: {} as SocketSession,
      maxPayloadLength: MAX_WS_PAYLOAD,
      backpressureLimit: 256 * 1024,
      closeOnBackpressureLimit: true,
      open(socket) {
        const session = socket.data
        if (closing) {
          void closeSession(session)
          socket.close(1012, "server shutting down")
          return
        }
        session.socket = socket
        status(socket, "connected")
      },
      message(socket, message) {
        const session = socket.data
        const byteLength = typeof message === "string" ? Buffer.byteLength(message) : message.byteLength
        if (!consumeInboundBudget(session, byteLength)) {
          socket.close(1008, "input rate limit")
          return
        }
        if (typeof message !== "string") {
          if (message.byteLength > MAX_INPUT_BYTES) {
            socket.close(1009, "payload too large")
            return
          }
          if (session.started && session.terminal && !session.closed) {
            try {
              session.terminal.write(message)
            } catch (error) {
              reportFailure(session, error)
              socket.close(1011, "terminal input failed")
            }
          }
          return
        }
        const control = parseTerminalControl(message)
        if (!control) {
          socket.close(1008, "invalid control")
          return
        }
        void enqueueSessionControl(session, () => handleControl(session, control)).catch((error) => reportFailure(session, error))
      },
      close(socket) {
        void closeSession(socket.data)
      },
    },
  })

  let closePromise: Promise<void> | undefined
  const close = () => {
    if (closePromise) return closePromise
    closing = true
    closePromise = (async () => {
      try {
        // `closing` rejects every later upgrade; sessions were registered
        // before upgrade, so this drains every connection that crossed it.
        const cleanupFailures = await drainSessions()
        let stopFailure: unknown
        try {
          await server.stop(true)
        } catch (error) {
          stopFailure = error
        }
        const failures = [...cleanupFailures]
        if (stopFailure) failures.push(stopFailure)
        if (failures.length > 0) throw new AggregateError(failures, "Terminal web shutdown did not cleanly dispose every session")
      } finally {
        process.removeListener("SIGINT", onSignal)
        process.removeListener("SIGTERM", onSignal)
      }
    })()
    return closePromise
  }
  const onSignal = () => {
    if (closing) return
    void close().then(
      () => process.exit(0),
      (error) => {
        console.error("stacks terminal lab shutdown failed", error)
        process.exit(1)
      },
    )
  }
  if (options.installSignalHandlers ?? true) {
    process.once("SIGINT", onSignal)
    process.once("SIGTERM", onSignal)
  }

  return {
    get url() { return server.url },
    get origin() { return server.url.origin },
    get activeSessionCount() { return sessions.size },
    get activeChildCount() {
      return [...sessions].reduce((count, session) => count + (session.child || session.pendingChildDisposals > 0 ? 1 : 0), 0)
    },
    close,
  }
}

if (import.meta.main) {
  const harness = await createTerminalWebServer()
  console.log(`stacks terminal lab → ${harness.url}`)
}
