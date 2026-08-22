package domain

import (
	"fmt"
	"strings"
)

// GetDisplayState resolves the single status that earns a PR's ball color and
// inspector headline. The order is product policy, not a rendering detail.
func GetDisplayState(pr PullRequest) PrDisplayState {
	if pr.Merged {
		return StateMerged
	}
	if pr.Draft {
		return StateDraft
	}
	if pr.CI.State == CIFailure || pr.CI.Failed > 0 {
		return StateCIFailure
	}

	reviewBlocked := pr.ChangesRequested ||
		pr.ReviewDecision == "CHANGES_REQUESTED" ||
		(pr.Mergeable != nil && !*pr.Mergeable)
	if reviewBlocked {
		return StateReviewBlocked
	}

	if pr.MergeQueueState != "" && pr.MergeQueueState != "NONE" {
		return StateQueued
	}
	if pr.ReviewDecision == "APPROVED" && pr.CI.State == CISuccess {
		return StateReady
	}
	return StateOpen
}

var DisplayStateLabel = map[PrDisplayState]string{
	StateMerged:        "merged",
	StateDraft:         "draft",
	StateCIFailure:     "CI failing",
	StateReviewBlocked: "review / merge blocked",
	StateQueued:        "merge queued",
	StateReady:         "ready to merge",
	StateOpen:          "open / pending",
}

func DisplayStateDetail(pr PullRequest) string {
	state := GetDisplayState(pr)
	switch state {
	case StateMerged:
		return "already landed"
	case StateDraft:
		return "not requesting review"
	case StateReviewBlocked:
		if pr.Mergeable != nil && !*pr.Mergeable {
			return "merge conflict"
		}
		if pr.ChangesRequested {
			return "changes requested"
		}
	case StateCIFailure:
		failed := pr.CI.Failed
		if failed == 0 {
			failed = 1
		}
		return fmt.Sprintf("%d CI check failed", failed)
	case StateQueued:
		return humanizeQueueState(pr.MergeQueueState)
	case StateReady:
		return "checks passed · review complete"
	}

	if pr.CI.State == CIPending && pr.Approvals > 0 {
		return "checks running · review complete"
	}
	if pr.CI.State == CIPending {
		return "checks running · awaiting review"
	}
	if pr.CI.State == CISuccess && pr.Approvals > 0 {
		return "checks passed · awaiting merge"
	}
	if pr.CI.State == CISuccess {
		return "checks passed · awaiting review"
	}
	if pr.Approvals > 0 {
		return "review complete · awaiting checks"
	}
	return "awaiting checks and review"
}

func humanizeQueueState(queueState string) string {
	if queueState == "" || queueState == "NONE" || queueState == "QUEUED" {
		return "waiting in merge queue"
	}
	lowered := strings.ToLower(queueState)
	lowered = strings.ReplaceAll(lowered, "_", " ")
	return strings.TrimPrefix(lowered, "merge queue ")
}

func StateColorToken(state PrDisplayState) string {
	switch state {
	case StateCIFailure:
		return "ciFailure"
	case StateReviewBlocked:
		return "reviewBlocked"
	default:
		return string(state)
	}
}
