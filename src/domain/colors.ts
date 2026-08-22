export type Oklch = readonly [lightness: number, chroma: number, hue: number]

const clamp = (value: number, lower = 0, upper = 1) => Math.min(upper, Math.max(lower, value))

/** Converts an authored OKLCH token to a terminal-safe #RRGGBB color. */
export function oklchToSrgb(token: Oklch): readonly [number, number, number] {
  const [lightness, chroma, hue] = token
  const radians = (hue * Math.PI) / 180
  const a = chroma * Math.cos(radians)
  const b = chroma * Math.sin(radians)

  const l = lightness + 0.3963377774 * a + 0.2158037573 * b
  const m = lightness - 0.1055613458 * a - 0.0638541728 * b
  const s = lightness - 0.0894841775 * a - 1.291485548 * b
  const l3 = l * l * l
  const m3 = m * m * m
  const s3 = s * s * s

  const linear = [
    4.0767416621 * l3 - 3.3077115913 * m3 + 0.2309699292 * s3,
    -1.2684380046 * l3 + 2.6097574011 * m3 - 0.3413193965 * s3,
    -0.0041960863 * l3 - 0.7034186147 * m3 + 1.707614701 * s3,
  ] as const

  const gamma = (channel: number) => {
    const clipped = clamp(channel)
    return clipped <= 0.0031308
      ? 12.92 * clipped
      : 1.055 * Math.pow(clipped, 1 / 2.4) - 0.055
  }

  return linear.map((channel) => Math.round(clamp(gamma(channel)) * 255)) as [number, number, number]
}

export function oklchToHex(token: Oklch): string {
  return `#${oklchToSrgb(token)
    .map((channel) => channel.toString(16).padStart(2, "0"))
    .join("")}`
}

/** Every visual color starts life as an OKLCH triple. */
export const oklchTokens = {
  surface: [0.18, 0.025, 260],
  surfaceRaised: [0.235, 0.03, 260],
  border: [0.37, 0.035, 260],
  text: [0.93, 0.018, 260],
  muted: [0.66, 0.026, 260],
  focus: [0.76, 0.13, 255],
  draft: [0.62, 0.025, 260],
  open: [0.67, 0.034, 260],
  queued: [0.8, 0.155, 88],
  ciFailure: [0.66, 0.19, 35],
  reviewBlocked: [0.6, 0.22, 19],
  ready: [0.72, 0.16, 145],
  merged: [0.65, 0.145, 305],
  success: [0.76, 0.13, 145],
  warning: [0.8, 0.14, 88],
} as const satisfies Record<string, Oklch>

export type ColorToken = keyof typeof oklchTokens

/** This is the only renderer boundary: components consume derived terminal hex. */
export const terminalColors: Record<ColorToken, string> = Object.fromEntries(
  Object.entries(oklchTokens).map(([name, token]) => [name, oklchToHex(token)]),
) as Record<ColorToken, string>

export const color = (token: ColorToken) => terminalColors[token]
