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
	if got := domain.GetDisplayState(merged); got != domain.StateMerged {
		t.Fatalf("merged+draft+ci failure: got %s", got)
	}

	draft := data.PRFixture(data.PullRequestFixtureInput{
		Draft: domain.BoolPtr(true),
		CI:    &domain.CISummary{State: domain.CIFailure, Failed: 2, Pending: 0, Total: 3},
	})
	if got := domain.GetDisplayState(draft); got != domain.StateDraft {
		t.Fatalf("draft+ci failure: got %s", got)
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
