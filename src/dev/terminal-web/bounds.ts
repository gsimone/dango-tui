export const TERMINAL_BOUNDS = {
  minCols: 2,
  maxCols: 500,
  minRows: 1,
  maxRows: 300,
} as const

export type TerminalSize = {
  cols: number
  rows: number
}

export function isTerminalSizeWithinBounds(value: unknown): value is TerminalSize {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return false
  const { cols, rows } = value as Record<string, unknown>
  return typeof cols === "number"
    && typeof rows === "number"
    && Number.isInteger(cols)
    && Number.isInteger(rows)
    && cols >= TERMINAL_BOUNDS.minCols
    && cols <= TERMINAL_BOUNDS.maxCols
    && rows >= TERMINAL_BOUNDS.minRows
    && rows <= TERMINAL_BOUNDS.maxRows
}

/** Clamp browser Fit proposals before they reach xterm or the PTY protocol. */
export function clampTerminalSize(size: TerminalSize): TerminalSize {
  return {
    cols: Math.min(TERMINAL_BOUNDS.maxCols, Math.max(TERMINAL_BOUNDS.minCols, Math.floor(size.cols))),
    rows: Math.min(TERMINAL_BOUNDS.maxRows, Math.max(TERMINAL_BOUNDS.minRows, Math.floor(size.rows))),
  }
}
