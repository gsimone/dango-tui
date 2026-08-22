import { describe, expect, test } from "bun:test"
import { testRender } from "@opentui/solid"
import { App } from "../app/App.tsx"
import { fixtureStories } from "../data/fixtures.ts"
import { clampCardPlacement, getBallPoint, getRowLayout, type TerminalSize } from "../components/layout.ts"

const makeUi = async (size: TerminalSize = { width: 80, height: 24 }, initialStoryId?: string) => {
  const ui = await testRender(() => <App mode="stories" initialStoryId={initialStoryId} />, size)
  await ui.flush()
  return ui
}

const assertFits = (frame: string, width: number) => {
  for (const line of frame.split("\n")) expect(line.length).toBeLessThanOrEqual(width)
}

describe("fixture app with OpenTUI testRender", () => {
  test("captures deterministic frames at compact, standard and wide widths", async () => {
    for (const size of [{ width: 40, height: 20 }, { width: 80, height: 24 }, { width: 120, height: 30 }] as const) {
      const ui = await makeUi(size)
      const frame = ui.captureCharFrame()
      expect(frame).toContain("STACKS UI LAB")
      expect(frame).toContain("mixed health")
      expect(frame).toContain("auth cleanup")
      expect(frame).toContain("●")
      assertFits(frame, size.width)
      ui.renderer.destroy()
    }
  })

  test("keyboard selection and hover reveal the exact same inspector card", async () => {
    const ui = await makeUi()
    await ui.mockInput.pressArrow("right")
    await ui.flush()
    const keyboardFrame = ui.captureCharFrame()
    expect(keyboardFrame).toContain("#185 Keep service identity explicit")
    expect(keyboardFrame).toContain("●—◉—●")

    const firstStack = fixtureStories[0]!.stacks[0]!
    const point = getBallPoint({ width: 80, height: 24 }, 0, 1, firstStack.prs.length)
    await ui.mockMouse.moveTo(point.x, point.y)
    await ui.flush()
    const hoverFrame = ui.captureCharFrame()
    expect(hoverFrame).toContain("#185 Keep service identity explicit")
    expect(hoverFrame).toContain("ready to merge")
    ui.renderer.destroy()
  })

  test("keyboard keeps tail/head selection and the compact card readable at 40 columns", async () => {
    const ui = await makeUi({ width: 40, height: 20 })
    await ui.mockInput.pressArrow("down")
    await ui.mockInput.pressKey("END")
    await ui.flush()
    expect(ui.captureCharFrame()).toContain("#213 Prepare token migration")
    await ui.mockInput.pressKey("HOME")
    await ui.flush()
    const compactFrame = ui.captureCharFrame()
    expect(compactFrame).toContain("#211 Add token catalogue")
    expect(compactFrame).toContain("click checkout · o open")
    expect(compactFrame).toContain("↑↓ stack · ←→ layer · / find · ? help")
    assertFits(compactFrame, 40)
    ui.renderer.destroy()
  })

  test("every layer owns a glyph plus connector/trailing blank mouse cell", async () => {
    const ui = await makeUi()
    const firstStack = fixtureStories[0]!.stacks[0]!
    const point = getBallPoint({ width: 80, height: 24 }, 0, 1, firstStack.prs.length)
    await ui.mockMouse.pressDown(point.x + 1, point.y)
    await ui.flush()
    expect(ui.captureCharFrame()).toContain("Checked out gm/stacks-185 · fixture simulation")
    const headPoint = getBallPoint({ width: 80, height: 24 }, 0, 2, firstStack.prs.length)
    await ui.mockMouse.pressDown(headPoint.x + 1, headPoint.y)
    await ui.flush()
    expect(ui.captureCharFrame()).toContain("Checked out gm/stacks-186 · fixture simulation")
    expect(getRowLayout(80, firstStack.prs.length).ballsWidth).toBe(firstStack.prs.length * 2)
    ui.renderer.destroy()
  })

  test("filters locally without touching fixture topology", async () => {
    const ui = await makeUi()
    ui.mockInput.pressKey("/")
    await ui.flush()
    await ui.mockInput.typeText("composer")
    await ui.flush()
    const frame = ui.captureCharFrame()
    expect(frame).toContain("Add ontology tokens to email")
    expect(frame).not.toContain("Simplify authentication boundaries")
    ui.renderer.destroy()
  })

  test("makes fixture cache state and no-data states explicit", async () => {
    const stale = await makeUi({ width: 80, height: 24 }, "draft")
    expect(stale.captureCharFrame()).toContain("fixture cache · stale (simulated)")
    stale.renderer.destroy()

    const empty = await makeUi({ width: 80, height: 24 }, "all-merged")
    expect(empty.captureCharFrame()).toContain("No open stacks in this fixture repository.")
    empty.renderer.destroy()

    const error = await makeUi({ width: 80, height: 24 }, "ci-failing")
    expect(error.captureCharFrame()).toContain("Refresh failed in this fixture. No cached stacks are available.")
    expect(error.captureCharFrame()).toContain("fixture refresh failed · no cached stacks")
    error.renderer.destroy()
  })

  test("search owns the footer and q becomes query text instead of quitting", async () => {
    const ui = await makeUi()
    ui.mockInput.pressKey("/")
    await ui.flush()
    const searchingFrame = ui.captureCharFrame()
    expect(searchingFrame).toContain("type to filter backspace edits esc clears / exits")
    expect(searchingFrame).not.toContain("q quit")

    await ui.mockInput.typeText("q")
    await ui.flush()
    expect(ui.renderer.isDestroyed).toBe(false)
    expect(ui.captureCharFrame()).toContain("q")
    ui.renderer.destroy()
  })

  test("keeps a compact, complete footer at 80 columns and a visible focused layer", async () => {
    const ui = await makeUi()
    const frame = ui.captureCharFrame()
    expect(frame).toContain("◉")
    expect(frame).toContain("? help · q quit")
    ui.renderer.destroy()
  })

  test("resizes through 40 to 160 columns and clamps the card against every edge", async () => {
    const ui = await makeUi({ width: 160, height: 40 })
    for (const size of [{ width: 40, height: 20 }, { width: 80, height: 24 }, { width: 120, height: 30 }, { width: 160, height: 40 }] as const) {
      ui.resize(size.width, size.height)
      await ui.flush()
      const frame = ui.captureCharFrame()
      expect(frame).toContain("STACKS UI LAB")
      assertFits(frame, size.width)
      for (const anchor of [{ x: 0, y: 0 }, { x: size.width - 1, y: 1 }, { x: size.width - 1, y: size.height - 3 }, { x: 1, y: size.height - 3 }]) {
        const placement = clampCardPlacement(size, anchor)
        expect(placement.left).toBeGreaterThanOrEqual(1)
        expect(placement.top).toBeGreaterThanOrEqual(1)
        expect(placement.left + placement.width).toBeLessThanOrEqual(size.width - 1)
        expect(placement.top + placement.height).toBeLessThanOrEqual(size.height - 2)
      }
    }
    const large = fixtureStories.find((story) => story.id === "large-stack")!.stacks[0]!
    const compact = getRowLayout(40, large.prs.length)
    expect(compact.compact).toBe(true)
    expect(compact.nameWidth + compact.ballsWidth).toBeLessThanOrEqual(38)
    ui.renderer.destroy()
  })

  test("q only tears down this renderer cleanly", async () => {
    const ui = await makeUi()
    ui.mockInput.pressKey("q")
    await ui.waitFor(() => ui.renderer.isDestroyed)
    expect(ui.renderer.isDestroyed).toBe(true)
  })
})
