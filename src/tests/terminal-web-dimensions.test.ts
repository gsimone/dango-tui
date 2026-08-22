import { describe, expect, test } from "bun:test"
import {
  DEFAULT_DIMENSION_MODE,
  dimensionsForContainerResize,
  isPresetActive,
  presetMode,
} from "../dev/terminal-web/dimensions.ts"
import { clampTerminalSize } from "../dev/terminal-web/bounds.ts"

describe("terminal web dimension policy", () => {
  test("the default and every exact preset ignore ResizeObserver's fitted dimensions", () => {
    const containerFit = { cols: 137, rows: 42 }
    const defaultResult = dimensionsForContainerResize(DEFAULT_DIMENSION_MODE, containerFit)
    expect(defaultResult.cols).toBe(80)
    expect(defaultResult.rows).toBe(24)

    for (const presetId of ["40x20", "80x24", "120x30", "160x40"] as const) {
      const mode = presetMode(presetId)
      if (mode.kind !== "preset") throw new Error("A preset control must create preset mode")
      expect(dimensionsForContainerResize(mode, containerFit)).toEqual(mode.preset)
      expect(isPresetActive(mode, presetId)).toBe(true)
      expect(isPresetActive(mode, "80x24")).toBe(presetId === "80x24")
    }
  })

  test("only explicit Fit mode accepts dimensions proposed by the container", () => {
    const fitted = { cols: 137, rows: 42 }
    expect(dimensionsForContainerResize({ kind: "fit" }, fitted)).toEqual(fitted)
    expect(isPresetActive({ kind: "fit" }, "80x24")).toBe(false)
  })

  test("Fit proposals are clamped before xterm can emit an out-of-protocol resize", () => {
    expect(clampTerminalSize({ cols: 912, rows: 742 })).toEqual({ cols: 500, rows: 300 })
    expect(clampTerminalSize({ cols: 1, rows: 0 })).toEqual({ cols: 2, rows: 1 })
  })
})
