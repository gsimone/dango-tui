package domain_test

import (
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

// Deterministic mutants. No math/rand, no shuffled cases.
// A mutant is killed when the real function is correct and the
// rewritten function is not, on a fixed input.

func TestMutantDisplayStateOrder(t *testing.T) {
	mergedDraft := domain.PullRequest{
		Merged: true,
		Draft:  true,
		CI:     domain.CISummary{State: domain.CIFailure, Failed: 1},
	}
	if got := domain.GetDisplayState(mergedDraft); got != domain.StateCIFailure {
		t.Fatalf("real fail-wins: %s", got)
	}
	mergedFirst := func(pr domain.PullRequest) domain.PrDisplayState {
		if pr.Merged {
			return domain.StateMerged
		}
		return domain.GetDisplayState(pr)
	}
	if mergedFirst(mergedDraft) == domain.StateCIFailure {
		t.Fatal("merged-before-fail mutant must not survive")
	}

	ciBeatsReview := domain.PullRequest{
		CI:               domain.CISummary{State: domain.CIFailure, Failed: 1},
		ChangesRequested: true,
	}
	if got := domain.GetDisplayState(ciBeatsReview); got != domain.StateCIFailure {
		t.Fatalf("real ci+blocked: %s", got)
	}
	reviewFirst := func(pr domain.PullRequest) domain.PrDisplayState {
		if pr.ChangesRequested {
			return domain.StateReviewBlocked
		}
		return domain.GetDisplayState(pr)
	}
	if reviewFirst(ciBeatsReview) == domain.StateCIFailure {
		t.Fatal("review-before-ci mutant must not survive")
	}

	conflict := domain.PullRequest{
		Mergeable:       domain.MergeableFalse(),
		MergeQueueState: "QUEUED",
		ReviewDecision:  "APPROVED",
		CI:              domain.CISummary{State: domain.CISuccess},
	}
	if got := domain.GetDisplayState(conflict); got != domain.StateReviewBlocked {
		t.Fatalf("real conflict: %s", got)
	}
	queueFirst := func(pr domain.PullRequest) domain.PrDisplayState {
		if pr.MergeQueueState != "" {
			return domain.StateQueued
		}
		return domain.GetDisplayState(pr)
	}
	if queueFirst(conflict) == domain.StateReviewBlocked {
		t.Fatal("queue-before-conflict mutant must not survive")
	}

	approvedQueued := domain.PullRequest{
		MergeQueueState: "QUEUED",
		ReviewDecision:  "APPROVED",
	}
	if got := domain.GetDisplayState(approvedQueued); got != domain.StateReady {
		t.Fatalf("real approved-over-queue: %s", got)
	}
	queueOverApproved := func(pr domain.PullRequest) domain.PrDisplayState {
		if pr.MergeQueueState != "" {
			return domain.StateQueued
		}
		return domain.GetDisplayState(pr)
	}
	if queueOverApproved(approvedQueued) == domain.StateReady {
		t.Fatal("queue-before-approved mutant must not survive")
	}
}

func TestMutantBallGlyphNeverFilledStatus(t *testing.T) {
	if domain.BallGlyph(domain.StateCIFailure) == '●' || domain.BallGlyph(domain.StateOpen) == '●' {
		t.Fatal("filled is not a status")
	}
	filledStatus := func(state domain.PrDisplayState) rune {
		if state == domain.StateReviewBlocked {
			return '◎'
		}
		if state == domain.StateQueued {
			return '◌'
		}
		return '●'
	}
	if filledStatus(domain.StateCIFailure) == domain.BallGlyph(domain.StateCIFailure) {
		t.Fatal("filled-as-status mutant must not survive")
	}
}

func TestMutantNormalizeHexAndLoginColor(t *testing.T) {
	if domain.NormalizeHex("D73A4A") != "#d73a4a" {
		t.Fatalf("real hex: %q", domain.NormalizeHex("D73A4A"))
	}
	keepCase := func(raw string) string {
		if len(raw) == 6 {
			return "#" + raw
		}
		return domain.NormalizeHex(raw)
	}
	if keepCase("D73A4A") == "#d73a4a" {
		t.Fatal("case-preserving mutant must not survive")
	}

	a := domain.LoginColor("gianni")
	if a != domain.LoginColor("gianni") || domain.IsLowChromaHex(a) {
		t.Fatalf("real login color %q", a)
	}
	alwaysMeta := func(string) string { return domain.Color("meta") }
	if alwaysMeta("gianni") == a {
		t.Fatal("meta-login mutant must not survive")
	}
	if domain.LoginColor("gianni") == domain.LoginColor("lina") {
		t.Fatal("different logins must not collide")
	}
}

func TestMutantStateColorToken(t *testing.T) {
	if domain.StateColorToken(domain.StateCIFailure) != "ciFailure" {
		t.Fatal("real CI token")
	}
	identity := func(state domain.PrDisplayState) string { return string(state) }
	if identity(domain.StateCIFailure) == "ciFailure" {
		t.Fatal("identity-token mutant must not survive")
	}
}
