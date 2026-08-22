import { FitAddon } from "@xterm/addon-fit"
import { Terminal } from "@xterm/xterm"
import {
  DEFAULT_DIMENSION_MODE,
  dimensionsForContainerResize,
  isPresetActive,
  presetById,
  presetMode,
  type DimensionMode,
} from "./dimensions.ts"
import { clampTerminalSize } from "./bounds.ts"

type StatusMessage = { type: "status"; status: "connected" | "started" | "reset" | "restarted" | "exited" | "error"; detail?: string }

const token = document.querySelector<HTMLMetaElement>('meta[name="stacks-terminal-token"]')?.content
const terminalRoot = document.querySelector<HTMLElement>("#terminal")
const storySelect = document.querySelector<HTMLSelectElement>("#story-select")
const restartButton = document.querySelector<HTMLButtonElement>("#restart")
const focusButton = document.querySelector<HTMLButtonElement>("#focus-terminal")
const statusElement = document.querySelector<HTMLElement>("#connection-status")
const presetButtons = [...document.querySelectorAll<HTMLButtonElement>("[data-preset]")]
const fitButton = document.querySelector<HTMLButtonElement>("#fit-terminal")

if (!token || !terminalRoot || !storySelect || !restartButton || !focusButton || !fitButton || !statusElement) {
  throw new Error("The terminal web harness markup is incomplete.")
}

const textEncoder = new TextEncoder()
const inputBytes = (data: string) => textEncoder.encode(data)
const binaryBytes = (data: string) => Uint8Array.from(data, (character) => character.charCodeAt(0))

const terminal = new Terminal({
  cursorBlink: true,
  cursorStyle: "bar",
  fontFamily: '"Departure Mono", "SFMono-Regular", Consolas, "Liberation Mono", monospace',
  fontSize: 14,
  lineHeight: 1.18,
  scrollback: 1_000,
  theme: {
    background: "oklch(18% 0.025 260)",
    foreground: "oklch(93% 0.018 260)",
    cursor: "oklch(76% 0.13 255)",
    selectionBackground: "oklch(37% 0.035 260)",
  },
})
const fitAddon = new FitAddon()
terminal.loadAddon(fitAddon)
terminal.open(terminalRoot)

let socket: WebSocket | undefined
let resizeTimer: ReturnType<typeof setTimeout> | undefined
let dimensionMode: DimensionMode = DEFAULT_DIMENSION_MODE
let ptyStarted = false

const setStatus = (state: StatusMessage["status"] | "connecting" | "closed", detail?: string) => {
  statusElement.dataset.state = state
  statusElement.textContent = detail ? `${state} · ${detail}` : state
}

const sendControl = (control: object) => {
  if (!socket || socket.readyState !== WebSocket.OPEN) return false
  socket.send(JSON.stringify(control))
  return true
}

const syncActiveDimensionControl = () => {
  for (const button of presetButtons) button.setAttribute("aria-pressed", String(isPresetActive(dimensionMode, button.dataset.preset)))
  fitButton.setAttribute("aria-pressed", String(dimensionMode.kind === "fit"))
}

const setTerminalDimensions = (cols: number, rows: number) => {
  if (terminal.cols !== cols || terminal.rows !== rows) terminal.resize(cols, rows)
}

const applyDimensionMode = () => {
  if (dimensionMode.kind === "fit") {
    const proposed = fitAddon.proposeDimensions()
    if (proposed) {
      const fitted = clampTerminalSize(proposed)
      setTerminalDimensions(fitted.cols, fitted.rows)
    }
    return
  }
  const exact = dimensionsForContainerResize(dimensionMode, { cols: terminal.cols, rows: terminal.rows })
  setTerminalDimensions(exact.cols, exact.rows)
}

const handleContainerResize = () => {
  // Exact presets deliberately ignore browser/container geometry. Only the
  // explicit Fit mode lets ResizeObserver determine terminal dimensions.
  applyDimensionMode()
}

const debounceContainerResize = () => {
  if (resizeTimer) clearTimeout(resizeTimer)
  resizeTimer = setTimeout(handleContainerResize, 48)
}

const setDimensionMode = (next: DimensionMode) => {
  dimensionMode = next
  syncActiveDimensionControl()
  applyDimensionMode()
  terminal.focus()
}

terminal.onResize(({ cols, rows }) => {
  if (ptyStarted && socket?.readyState === WebSocket.OPEN) sendControl({ type: "resize", cols, rows })
})
terminal.onData((data) => {
  if (socket?.readyState === WebSocket.OPEN) socket.send(inputBytes(data))
})
terminal.onBinary((data) => {
  if (socket?.readyState === WebSocket.OPEN) socket.send(binaryBytes(data))
})

syncActiveDimensionControl()
new ResizeObserver(debounceContainerResize).observe(terminalRoot)
terminalRoot.addEventListener("pointerdown", () => terminal.focus())

for (const button of presetButtons) {
  button.addEventListener("click", () => {
    const preset = presetById(button.dataset.preset)
    if (preset) setDimensionMode(presetMode(preset.id))
  })
}

fitButton.addEventListener("click", () => setDimensionMode({ kind: "fit" }))
focusButton.addEventListener("click", () => terminal.focus())
restartButton.addEventListener("click", () => {
  sendControl({ type: "restart", storyId: storySelect.value, cols: terminal.cols, rows: terminal.rows })
  terminal.focus()
})

storySelect.addEventListener("change", () => {
  sendControl({ type: "restart", storyId: storySelect.value, cols: terminal.cols, rows: terminal.rows })
  terminal.focus()
})

const handleIncoming = async (event: MessageEvent) => {
  if (typeof event.data === "string") {
    try {
      const message = JSON.parse(event.data) as StatusMessage
      if (message.type === "status") {
        if (message.status === "reset") {
          // Ordered server control frame: purge the previous native alt-screen
          // before the replacement PTY can write a byte.
          terminal.reset()
          terminal.clear()
        }
        setStatus(message.status, message.detail)
      }
    } catch {
      // The server never sends terminal output as text, but a malformed status is non-fatal.
    }
    return
  }
  if (event.data instanceof ArrayBuffer) {
    terminal.write(new Uint8Array(event.data))
    return
  }
  if (event.data instanceof Blob) terminal.write(new Uint8Array(await event.data.arrayBuffer()))
}

const connect = () => {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:"
  socket = new WebSocket(`${protocol}//${location.host}/pty?token=${encodeURIComponent(token)}`)
  socket.binaryType = "arraybuffer"
  setStatus("connecting")
  socket.addEventListener("open", () => {
    setStatus("connected")
    // The default starts exactly 80×24. Fit is an explicit opt-in mode.
    ptyStarted = false
    applyDimensionMode()
    sendControl({ type: "start", storyId: storySelect.value, cols: terminal.cols, rows: terminal.rows })
    ptyStarted = true
    terminal.focus()
  })
  socket.addEventListener("message", (event) => void handleIncoming(event))
  socket.addEventListener("close", (event) => {
    ptyStarted = false
    setStatus("closed", event.reason || `code ${event.code}`)
  })
  socket.addEventListener("error", () => setStatus("error", "socket failure"))
}

connect()
