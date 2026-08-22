import type { PrDisplayState, PullRequest } from "./types.ts"

/**
 * Resolve the single status that earns a PR's ball color and inspector headline.
 * The order is intentionally product policy, not a rendering detail.
 */
export function getDisplayState(pr: PullRequest): PrDisplayState {
  if (pr.merged) return "merged"
  if (pr.draft) return "draft"
  if (pr.ci.state === "failure" || pr.ci.failed > 0) return "ci-failure"

  const reviewBlocked =
    pr.changesRequested ||
    pr.reviewDecision === "CHANGES_REQUESTED" ||
    pr.mergeable === false
  if (reviewBlocked) return "review-blocked"

  if (pr.mergeQueueState && pr.mergeQueueState !== "NONE") return "queued"
  if (pr.reviewDecision === "APPROVED" && pr.ci.state === "success") return "ready"
  return "open"
}

export const displayStateLabel: Record<PrDisplayState, string> = {
  merged: "merged",
  draft: "draft",
  "ci-failure": "CI failing",
  "review-blocked": "review / merge blocked",
  queued: "merge queued",
  ready: "ready to merge",
  open: "open / pending",
}

export function displayStateDetail(pr: PullRequest): string {
  const state = getDisplayState(pr)
  if (state === "merged") return "already landed"
  if (state === "draft") return "not requesting review"
  if (state === "review-blocked" && pr.mergeable === false) return "merge conflict"
  if (state === "review-blocked" && pr.changesRequested) return "changes requested"
  if (state === "ci-failure") return `${pr.ci.failed || 1} CI check failed`
  if (state === "queued") return humanizeQueueState(pr.mergeQueueState)
  if (state === "ready") return "checks passed · review complete"
  if (pr.ci.state === "pending" && pr.approvals > 0) return "checks running · review complete"
  if (pr.ci.state === "pending") return "checks running · awaiting review"
  if (pr.ci.state === "success" && pr.approvals > 0) return "checks passed · awaiting merge"
  if (pr.ci.state === "success") return "checks passed · awaiting review"
  if (pr.approvals > 0) return "review complete · awaiting checks"
  return "awaiting checks and review"
}

function humanizeQueueState(queueState: string | undefined): string {
  if (!queueState || queueState === "NONE" || queueState === "QUEUED") return "waiting in merge queue"
  return queueState
    .toLowerCase()
    .replaceAll("_", " ")
    .replace(/^merge queue /, "")
}
