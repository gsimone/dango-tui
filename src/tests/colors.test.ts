import { describe, expect, test } from "bun:test"
import { oklchToHex, oklchToSrgb, oklchTokens, terminalColors } from "../domain/colors.ts"

describe("OKLCH terminal palette", () => {
  test("converts neutral endpoints and produces valid terminal colors", () => {
    expect(oklchToSrgb([0, 0, 0])).toEqual([0, 0, 0])
    expect(oklchToSrgb([1, 0, 0])).toEqual([255, 255, 255])
    expect(oklchToHex(oklchTokens.ready)).toMatch(/^#[0-9a-f]{6}$/)
    for (const value of Object.values(terminalColors)) expect(value).toMatch(/^#[0-9a-f]{6}$/)
  })

  test("keeps near-neutral open/draft close and danger states visibly distinct", () => {
    const [draftL, draftC, draftHue] = oklchTokens.draft
    const [openL, openC, openHue] = oklchTokens.open
    expect(Math.abs(draftL - openL)).toBeLessThan(0.08)
    expect(Math.abs(draftC - openC)).toBeLessThan(0.02)
    expect(Math.abs(draftHue - openHue)).toBeLessThan(1)
    expect(Math.abs(oklchTokens.ciFailure[2] - oklchTokens.reviewBlocked[2])).toBeGreaterThan(10)
    expect(oklchTokens.queued[2]).toBeGreaterThan(70)
    expect(oklchTokens.ready[1]).toBeGreaterThan(0.1)
    expect(oklchTokens.merged[2]).toBeGreaterThan(280)
  })
})
