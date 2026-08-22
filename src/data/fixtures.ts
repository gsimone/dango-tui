import type { PrDisplayState, PullRequest, Stack } from "../domain/types.ts"

export type PullRequestFixtureInput = Partial<PullRequest> & {
  state?: PrDisplayState
  number?: number
}

const stateFlags: Record<PrDisplayState, Partial<PullRequest>> = {
  merged: { merged: true },
  draft: { draft: true },
  "ci-failure": { ci: { state: "failure", failed: 1, pending: 0, total: 9 } },
  "review-blocked": { changesRequested: true, reviewDecision: "CHANGES_REQUESTED" },
  queued: { mergeQueueState: "QUEUED", reviewDecision: "APPROVED", ci: { state: "success", failed: 0, pending: 0, total: 9 } },
  ready: { reviewDecision: "APPROVED", approvals: 2, ci: { state: "success", failed: 0, pending: 0, total: 9 } },
  open: { ci: { state: "pending", failed: 0, pending: 2, total: 9 } },
}

export function prFixture(input: PullRequestFixtureInput = {}): PullRequest {
  const { state = "open", ci: ciOverride, ...overrides } = input
  const number = overrides.number ?? 100
  const flags = stateFlags[state]
  return {
    number,
    title: `Improve PR layer ${number}`,
    url: `https://github.com/example/stacks/pull/${number}`,
    branch: `gm/stacks-${number}`,
    author: "gianni",
    draft: false,
    merged: false,
    mergeable: true,
    mergeQueueState: undefined,
    reviewDecision: undefined,
    approvals: 0,
    changesRequested: false,
    additions: 43,
    deletions: 12,
    changedFiles: 4,
    headSha: `fixture${number.toString(16).padStart(8, "0")}`,
    ...flags,
    ...overrides,
    ci: ciOverride ?? flags.ci ?? { state: "unknown", failed: 0, pending: 0, total: 0 },
  }
}

export function stackFixture(input: Partial<Stack> & { number?: number; prs?: PullRequest[] } = {}): Stack {
  const number = input.number ?? 1
  return {
    id: input.id ?? `fixture-stack-${number}`,
    number,
    name: input.name ?? `stack ${number}`,
    baseRef: input.baseRef ?? "main",
    prs: input.prs ?? [prFixture({ number: number * 100 })],
    description: input.description ?? "A deterministic fixture stack",
    ...input,
  }
}

export type FixtureStory = {
  id: string
  label: string
  stacks: Stack[]
  cacheState?: "current" | "stale" | "error"
}

const pr = (number: number, title: string, state: PrDisplayState, extra: PullRequestFixtureInput = {}) =>
  prFixture({ number, title, state, ...extra })

export const fixtureStories: FixtureStory[] = [
  {
    id: "mixed",
    label: "mixed health",
    stacks: [
      stackFixture({ number: 1, name: "auth cleanup", description: "Simplify authentication boundaries", prs: [pr(184, "Split auth scope from session checks", "merged"), pr(185, "Keep service identity explicit", "ready"), pr(186, "Remove implicit session fallback", "ci-failure")] }),
      stackFixture({ number: 2, name: "composer tokens", description: "Add ontology tokens to email", prs: [pr(211, "Add token catalogue", "ready"), pr(212, "Map entity fields into composer", "review-blocked"), pr(213, "Prepare token migration", "queued")] }),
      stackFixture({ number: 3, name: "sync rewrite", description: "Fix optimistic sync lifecycle", prs: [pr(241, "Mark syncing work clearly", "draft"), pr(242, "Avoid duplicate invalidation", "open")] }),
    ],
  },
  { id: "all-ready", label: "all ready", stacks: [stackFixture({ number: 4, name: "profile editor", description: "Ship the profile editing stack", prs: [pr(301, "Add profile capability", "ready"), pr(302, "Wire profile editor", "ready"), pr(303, "Polish confirmation copy", "ready")] })] },
  { id: "all-merged", label: "empty repository", stacks: [] },
  { id: "draft", label: "stale cache", cacheState: "stale", stacks: [stackFixture({ number: 6, name: "release notes", description: "Cached 18m ago · waiting to refresh", prs: [pr(321, "Draft release notes from merged work", "open", { ci: { state: "unknown", failed: 0, pending: 0, total: 0 } }), pr(322, "Publish the rollout note", "ready")] })] },
  { id: "queued", label: "queued", stacks: [stackFixture({ number: 7, name: "release handoff", description: "Waiting in merge queue", prs: [pr(331, "Queue after release freeze", "queued")] })] },
  { id: "ci-failing", label: "refresh error", cacheState: "error", stacks: [] },
  { id: "changes-requested", label: "changes requested", stacks: [stackFixture({ number: 9, name: "checkout safety", description: "Needs one review follow-up", prs: [pr(351, "Protect local checkout action", "review-blocked", { approvals: 1, changesRequested: true })] })] },
  { id: "merge-conflict", label: "merge conflict", stacks: [stackFixture({ number: 10, name: "stack base", description: "Conflict against main", prs: [pr(361, "Rework stack base references", "review-blocked", { mergeable: false, changesRequested: false, reviewDecision: undefined })] })] },
  { id: "pending-no-review", label: "pending, no review", stacks: [stackFixture({ number: 11, name: "first review", description: "Awaiting CI and first review", prs: [pr(371, "Observe no review data", "open", { ci: { state: "pending", failed: 0, pending: 4, total: 9 }, mergeable: null, author: "lina" })] })] },
  { id: "long-title-branch", label: "long title + branch", stacks: [stackFixture({ number: 12, name: "terminal dignity", description: "Ensure narrow terminals truncate with dignity", prs: [pr(381, "Refactor the very long composition pipeline so entity token expansion remains deterministic when a deeply nested relationship resolves to an empty value", "ready", { branch: "gm/refactor-composer-token-expansion-for-deeply-nested-relationship-fallbacks" }), pr(382, "Document the branch naming experiment", "open", { branch: "gm/this-branch-name-is-deliberately-absurdly-long-to-test-the-card" })] })] },
  { id: "large-stack", label: "large stack", stacks: [stackFixture({ number: 13, name: "migration train", description: "Ten independent layers, still one readable line", prs: ["merged", "merged", "ready", "ready", "queued", "open", "draft", "ci-failure", "review-blocked", "open"].map((state, index) => pr(400 + index, `Large stack layer ${index + 1}`, state as PrDisplayState)) })] },
]

/** The browser PTY harness may only launch these deterministic fixture stories. */
export const fixtureStoryIds = fixtureStories.map((story) => story.id) as readonly string[]

export function isFixtureStoryId(value: unknown): value is string {
  return typeof value === "string" && fixtureStoryIds.includes(value)
}

export const storyById = (id: string) => fixtureStories.find((story) => story.id === id) ?? fixtureStories[0]!
