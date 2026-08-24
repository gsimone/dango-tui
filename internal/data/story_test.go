package data_test

import (
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
)

func TestDemoStoriesStayAuthored(t *testing.T) {
	for _, id := range []string{"mixed", "freight", "pair"} {
		story := data.StoryByID(id)
		if story.ID != id {
			t.Fatalf("missing story %s", id)
		}
		if len(story.Stacks) == 0 {
			t.Fatalf("%s must have stacks", id)
		}
		for _, stack := range story.Stacks {
			if strings.TrimSpace(stack.Name) == "" {
				t.Fatalf("%s has an unnamed stack", id)
			}
			for _, pr := range stack.PRs {
				if strings.TrimSpace(pr.Title) == "" {
					t.Fatalf("%s / %s has an untitled PR", id, stack.Name)
				}
				if strings.HasPrefix(pr.Title, "Freight layer") || strings.HasPrefix(pr.Title, "Tiny ") {
					t.Fatalf("%s still has filler titles: %q", id, pr.Title)
				}
			}
		}
	}

	mixed := data.StoryByID("mixed")
	if len(mixed.Stacks) != 3 || mixed.Stacks[0].Name != "auth cleanup" {
		t.Fatalf("mixed: %+v", mixed.Stacks)
	}
	if domain.GetDisplayState(mixed.Stacks[0].PRs[2]) != domain.StateCIFailure {
		t.Fatal("mixed keeps CI failing on auth head")
	}

	pair := data.StoryByID("pair")
	if len(pair.Stacks) != 1 || len(pair.Stacks[0].PRs) != 2 {
		t.Fatalf("pair: %+v", pair.Stacks)
	}
	if pair.Stacks[0].PRs[0].Title != "Land the checkout helper" {
		t.Fatalf("pair titles: %+v", pair.Stacks[0].PRs)
	}

	freight := data.StoryByID("freight")
	if len(freight.Stacks) != 1 || len(freight.Stacks[0].PRs) != 20 {
		t.Fatalf("freight layers: %d", len(freight.Stacks[0].PRs))
	}
	if freight.Stacks[0].PRs[0].Title != "Land the schema cutover" {
		t.Fatalf("freight titles: %q", freight.Stacks[0].PRs[0].Title)
	}
	seen := map[domain.PrDisplayState]bool{}
	for _, pr := range freight.Stacks[0].PRs {
		seen[domain.GetDisplayState(pr)] = true
	}
	for _, want := range []domain.PrDisplayState{domain.StateMerged, domain.StateReady, domain.StateCIFailure, domain.StateReviewBlocked} {
		if !seen[want] {
			t.Fatalf("freight missing state %s", want)
		}
	}
}

func TestUnknownStoryFallsBackToMixed(t *testing.T) {
	if data.StoryByID("missing").ID != "mixed" {
		t.Fatalf("unknown story should fall back to mixed, got %s", data.StoryByID("missing").ID)
	}
	if data.IsFixtureStoryID("chaos") || data.StoryByID("chaos").ID == "chaos" {
		t.Fatal("chaos must not ship in the binary")
	}
	for _, id := range []string{"all-ready", "queued", "merge-conflict", "pending-no-review", "long-title-branch"} {
		if data.IsFixtureStoryID(id) {
			t.Fatalf("%s is unused at runtime and must not ship", id)
		}
	}
}
