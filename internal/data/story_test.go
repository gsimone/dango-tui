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

func TestChaosStaysStressOnly(t *testing.T) {
	story := data.StoryByID("chaos")
	if len(story.Stacks) != 300 {
		t.Fatalf("chaos is the 300-stack stress story, got %d", len(story.Stacks))
	}
	if data.StoryByID("missing").ID != "mixed" {
		t.Fatalf("unknown story should fall back to mixed, got %s", data.StoryByID("missing").ID)
	}
}

func TestExampleStacksAreAuthoredNotChaos(t *testing.T) {
	stacks := data.ExampleStacks()
	if len(stacks) != 5 {
		t.Fatalf("mixed+pair+freight is 5 stacks, got %d", len(stacks))
	}
	layers := 0
	for _, stack := range stacks {
		layers += len(stack.PRs)
	}
	if layers != 30 {
		t.Fatalf("8+2+20 layers, got %d", layers)
	}
	names := strings.Join([]string{stacks[0].Name, stacks[3].Name, stacks[4].Name}, ",")
	if stacks[0].Name != "auth cleanup" || stacks[3].Name != "pair" || stacks[4].Name != "freight train" {
		t.Fatalf("example order: %s", names)
	}
	for _, stack := range stacks {
		for _, pr := range stack.PRs {
			if strings.HasPrefix(pr.Title, "Freight layer") || strings.Contains(pr.Title, " layer ") && strings.HasPrefix(stack.Name, "ok") {
				t.Fatalf("examples must stay authored: %q / %q", stack.Name, pr.Title)
			}
		}
	}
}
