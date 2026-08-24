package domain_test

import (
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
)

func TestGetDisplayStatePrecedence(t *testing.T) {
	merged := data.PRFixture(data.PullRequestFixtureInput{
		Merged: domain.BoolPtr(true),
		Draft:  domain.BoolPtr(true),
		CI:     &domain.CISummary{State: domain.CIFailure, Failed: 2, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(merged); got != domain.StateCIFailure {
		t.Fatalf("fail wins over merged: got %s", got)
	}

	draft := data.PRFixture(data.PullRequestFixtureInput{
		Draft: domain.BoolPtr(true),
		CI:    &domain.CISummary{State: domain.CIFailure, Failed: 2, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(draft); got != domain.StateCIFailure {
		t.Fatalf("fail wins over draft: got %s", got)
	}

	ci := data.PRFixture(data.PullRequestFixtureInput{
		CI:               &domain.CISummary{State: domain.CIFailure, Failed: 1, Pending: 0, Total: 3},
		ChangesRequested: domain.BoolPtr(true),
	})
	if got := domain.GetDisplayState(ci); got != domain.StateCIFailure {
		t.Fatalf("ci failure beats review: got %s", got)
	}

	blocked := data.PRFixture(data.PullRequestFixtureInput{
		ChangesRequested: domain.BoolPtr(true),
		MergeQueueState:  strPtr("QUEUED"),
		ReviewDecision:   strPtr("APPROVED"),
		CI:               &domain.CISummary{State: domain.CISuccess, Failed: 0, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(blocked); got != domain.StateReviewBlocked {
		t.Fatalf("changes requested beats queue: got %s", got)
	}

	conflict := data.PRFixture(data.PullRequestFixtureInput{
		Mergeable:       domain.MergeableFalse(),
		MergeQueueState: strPtr("QUEUED"),
		ReviewDecision:  strPtr("APPROVED"),
		CI:              &domain.CISummary{State: domain.CISuccess, Failed: 0, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(conflict); got != domain.StateReviewBlocked {
		t.Fatalf("unmergeable beats queue: got %s", got)
	}

	queued := data.PRFixture(data.PullRequestFixtureInput{
		MergeQueueState: strPtr("QUEUED"),
		ReviewDecision:  strPtr("APPROVED"),
		CI:              &domain.CISummary{State: domain.CISuccess, Failed: 0, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(queued); got != domain.StateQueued {
		t.Fatalf("queued: got %s", got)
	}

	ready := data.PRFixture(data.PullRequestFixtureInput{
		ReviewDecision: strPtr("APPROVED"),
		CI:             &domain.CISummary{State: domain.CISuccess, Failed: 0, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(ready); got != domain.StateReady {
		t.Fatalf("ready: got %s", got)
	}

	if got := domain.GetDisplayState(data.PRFixture(data.PullRequestFixtureInput{})); got != domain.StateOpen {
		t.Fatalf("default: got %s", got)
	}
}

func TestStateColorTokenFromPRFields(t *testing.T) {
	cases := []struct {
		name  string
		pr    domain.PullRequest
		token string
	}{
		{name: "draft", pr: domain.PullRequest{Draft: true}, token: "meta"},
		{name: "merged", pr: domain.PullRequest{Merged: true, Draft: true}, token: "merged"},
		{name: "blocked-review", pr: domain.PullRequest{ReviewDecision: "CHANGES_REQUESTED"}, token: "warning"},
		{name: "blocked-conflict", pr: domain.PullRequest{Mergeable: domain.MergeableFalse()}, token: "warning"},
		{name: "open-slim", pr: domain.PullRequest{}, token: "paper"},
		{name: "open-isDraft-state-only", pr: domain.PullRequest{Draft: false, Merged: false}, token: "paper"},
		{name: "queued", pr: domain.PullRequest{MergeQueueState: "QUEUED"}, token: "queued"},
		{name: "approved-no-ci", pr: domain.PullRequest{ReviewDecision: "APPROVED"}, token: "ready"},
		{name: "fail-wins", pr: domain.PullRequest{Draft: true, CI: domain.CISummary{State: domain.CIFailure, Failed: 1}}, token: "ciFailure"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := domain.GetDisplayState(tc.pr)
			if got := domain.StateColorToken(state); got != tc.token {
				t.Fatalf("token %q, want %q (state %s)", got, tc.token, state)
			}
		})
	}
}

func TestBallGlyphsAreOneMeaning(t *testing.T) {
	cases := []struct {
		state domain.PrDisplayState
		glyph rune
	}{
		{domain.StateOpen, '●'},
		{domain.StateDraft, '○'},
		{domain.StateCIFailure, '●'},
		{domain.StateReviewBlocked, '◎'},
		{domain.StateReady, '●'},
		{domain.StateMerged, '●'},
		{domain.StateQueued, '◌'},
	}
	for _, tc := range cases {
		if got := domain.BallGlyph(tc.state); got != tc.glyph {
			t.Fatalf("%s glyph %q, want %q", tc.state, string(got), string(tc.glyph))
		}
		if domain.IsLogoToken(domain.StateColorToken(tc.state)) {
			t.Fatalf("%s used a logo token", tc.state)
		}
	}
}

func TestDisplayStateHeadlines(t *testing.T) {
	for _, story := range data.Stories() {
		for _, stack := range story.Stacks {
			for _, pr := range stack.PRs {
				label := strings.ToLower(domain.DisplayStateLabel[domain.GetDisplayState(pr)])
				detail := strings.ToLower(domain.DisplayStateDetail(pr))
				headline := label + " · " + detail
				if detail == label {
					t.Fatalf("detail duplicates label for #%d: %s", pr.Number, headline)
				}
				if strings.Contains(headline, label+" · "+label) {
					t.Fatalf("duplicative headline for #%d: %s", pr.Number, headline)
				}
			}
		}
	}
}

func strPtr(v string) *string { return &v }
