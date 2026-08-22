import { isFixtureStoryId } from "../../data/fixtures.ts"
import { isTerminalSizeWithinBounds, type TerminalSize } from "./bounds.ts"

export { TERMINAL_BOUNDS, type TerminalSize } from "./bounds.ts"

export const MAX_CONTROL_BYTES = 8 * 1024
export const MAX_INPUT_BYTES = 64 * 1024

export type StartControl = TerminalSize & {
  type: "start"
  storyId: string
}

export type ResizeControl = TerminalSize & {
  type: "resize"
}

export type RestartControl = TerminalSize & {
  type: "restart"
  storyId: string
}

export type TerminalControl = StartControl | ResizeControl | RestartControl

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null && !Array.isArray(value)

const hasExactKeys = (value: Record<string, unknown>, keys: readonly string[]) => {
  const actual = Object.keys(value).sort()
  const expected = [...keys].sort()
  return actual.length === expected.length && actual.every((key, index) => key === expected[index])
}

export function isValidTerminalSize(value: unknown): value is TerminalSize {
  return isTerminalSizeWithinBounds(value)
}

/**
 * Accept only the three small bridge controls. Terminal input is always binary,
 * never smuggled through a JSON command channel.
 */
export function parseTerminalControl(raw: string): TerminalControl | null {
  if (new TextEncoder().encode(raw).byteLength > MAX_CONTROL_BYTES) return null

  let value: unknown
  try {
    value = JSON.parse(raw)
  } catch {
    return null
  }

  if (!isRecord(value) || typeof value.type !== "string") return null

  if (value.type === "resize") {
    return hasExactKeys(value, ["type", "cols", "rows"]) && isValidTerminalSize(value)
      ? { type: "resize", cols: value.cols, rows: value.rows }
      : null
  }

  if (value.type === "start" || value.type === "restart") {
    const type = value.type
    const storyId = value.storyId
    return hasExactKeys(value, ["type", "storyId", "cols", "rows"])
      && isValidTerminalSize(value)
      && isFixtureStoryId(storyId)
      ? { type, storyId, cols: value.cols, rows: value.rows }
      : null
  }

  return null
}
