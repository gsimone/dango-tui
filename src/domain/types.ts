export type CiState = "success" | "pending" | "failure" | "unknown"

export type CiSummary = {
  state: CiState
  failed: number
  pending: number
  total: number
}

export type PullRequest = {
  number: number
  title: string
  url: string
  branch: string
  author: string
  draft: boolean
  merged: boolean
  mergeable: boolean | null
  mergeQueueState?: string
  reviewDecision?: string
  approvals: number
  changesRequested: boolean
  ci: CiSummary
  additions: number
  deletions: number
  changedFiles: number
  headSha: string
}

export type Stack = {
  id: string
  number: number
  name: string
  baseRef: string
  prs: PullRequest[]
  description?: string
}

export type PrDisplayState =
  | "merged"
  | "draft"
  | "ci-failure"
  | "review-blocked"
  | "queued"
  | "ready"
  | "open"
