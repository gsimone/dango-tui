import { describe, expect, test } from "bun:test"
import { displayStateDetail, displayStateLabel, getDisplayState } from "../domain/displayState.ts"
import { fixtureStories, prFixture } from "../data/fixtures.ts"

describe("getDisplayState", () => {
  test("uses the published precedence, not the first convenient boolean", () => {
    expect(getDisplayState(prFixture({ merged: true, draft: true, ci: { state: "failure", failed: 2, pending: 0, total: 3 } }))).toBe("merged")
    expect(getDisplayState(prFixture({ draft: true, ci: { state: "failure", failed: 2, pending: 0, total: 3 } }))).toBe("draft")
    expect(getDisplayState(prFixture({ ci: { state: "failure", failed: 1, pending: 0, total: 3 }, changesRequested: true }))).toBe("ci-failure")
    expect(getDisplayState(prFixture({ changesRequested: true, mergeQueueState: "QUEUED", reviewDecision: "APPROVED", ci: { state: "success", failed: 0, pending: 0, total: 3 } }))).toBe("review-blocked")
    expect(getDisplayState(prFixture({ mergeable: false, mergeQueueState: "QUEUED", reviewDecision: "APPROVED", ci: { state: "success", failed: 0, pending: 0, total: 3 } }))).toBe("review-blocked")
    expect(getDisplayState(prFixture({ mergeQueueState: "QUEUED", reviewDecision: "APPROVED", ci: { state: "success", failed: 0, pending: 0, total: 3 } }))).toBe("queued")
    expect(getDisplayState(prFixture({ reviewDecision: "APPROVED", ci: { state: "success", failed: 0, pending: 0, total: 3 } }))).toBe("ready")
    expect(getDisplayState(prFixture())).toBe("open")
  })

  test("writes explanatory, non-duplicative headlines for every fixture state", () => {
    for (const story of fixtureStories) {
      for (const stack of story.stacks) {
        for (const pr of stack.prs) {
          const label = displayStateLabel[getDisplayState(pr)].toLowerCase()
          const detail = displayStateDetail(pr).toLowerCase()
          const headline = `${label} · ${detail}`
          expect(detail).not.toBe(label)
          expect(headline).not.toContain(`${label} · ${label}`)
        }
      }
    }
  })
})
