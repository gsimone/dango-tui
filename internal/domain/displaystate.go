package domain

import (
	"fmt"
	"strings"
)

// GetDisplayState resolves the single status that earns a PR's ball.
// Worst state wins: fail > review > approved > open > draft > queued > merged.
func GetDisplayState(pr PullRequest) PrDisplayState {
	if pr.CI.State == CIFailure || pr.CI.Failed > 0 {
		return StateCIFailure
	}

	reviewBlocked := pr.ChangesRequested ||
		pr.ReviewDecision == "CHANGES_REQUESTED" ||
		(pr.Mergeable != nil && !*pr.Mergeable)
	if reviewBlocked {
		return StateReviewBlocked
	}

	if pr.ReviewDecision == "APPROVED" {
		return StateReady
	}
	if pr.Draft {
		return StateDraft
	}
	if pr.MergeQueueState != "" && pr.MergeQueueState != "NONE" {
		return StateQueued
	}
	if pr.Merged {
		return StateMerged
	}
	return StateOpen
}

var DisplayStateLabel = map[PrDisplayState]string{
	StateMerged:        "merged",
	StateDraft:         "draft",
	StateCIFailure:     "CI failing",
	StateReviewBlocked: "needs your review",
	StateQueued:        "merge queued",
	StateReady:         "approved",
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
		return "warning"
	case StateDraft:
		return "draft"
	case StateReady:
		return "ready"
	case StateMerged:
		return "merged"
	case StateQueued, StateOpen:
		return "paper"
	default:
		return "paper"
	}
}

// BallGlyph is the resting mark. Filled is never a status — the list
// paints filled only on the active layer. Review is double, queued is dotted,
// everything else is hollow. No logo rainbow.
func BallGlyph(state PrDisplayState) rune {
	switch state {
	case StateReviewBlocked:
		return '◎'
	case StateQueued:
		return '◌'
	default:
		return '○'
	}
}
