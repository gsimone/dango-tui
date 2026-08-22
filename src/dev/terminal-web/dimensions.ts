export type TerminalDimensions = {
  cols: number
  rows: number
}

export const TERMINAL_PRESETS = [
  { id: "40x20", cols: 40, rows: 20 },
  { id: "80x24", cols: 80, rows: 24 },
  { id: "120x30", cols: 120, rows: 30 },
  { id: "160x40", cols: 160, rows: 40 },
] as const satisfies readonly (TerminalDimensions & { id: string })[]

export type TerminalPreset = (typeof TERMINAL_PRESETS)[number]
export type TerminalPresetId = TerminalPreset["id"]

export type DimensionMode =
  | { kind: "preset"; preset: TerminalPreset }
  | { kind: "fit" }

export const DEFAULT_DIMENSION_MODE: DimensionMode = {
  kind: "preset",
  preset: TERMINAL_PRESETS[1]!,
}

export function presetById(id: string | undefined): TerminalPreset | undefined {
  return TERMINAL_PRESETS.find((preset) => preset.id === id)
}

export function presetMode(id: TerminalPresetId): DimensionMode {
  const preset = presetById(id)
  if (!preset) throw new Error(`Unknown terminal preset: ${id}`)
  return { kind: "preset", preset }
}

/**
 * Container size only matters in Fit mode. This is the policy boundary that
 * keeps an active exact-preset label truthful across ResizeObserver events.
 */
export function dimensionsForContainerResize(mode: DimensionMode, fitted: TerminalDimensions): TerminalDimensions {
  return mode.kind === "preset" ? mode.preset : fitted
}

export function isPresetActive(mode: DimensionMode, id: string | undefined) {
  return mode.kind === "preset" && mode.preset.id === id
}
