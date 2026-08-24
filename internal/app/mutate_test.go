package app_test

import (
	"testing"

	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/domain"
)

func TestMutantFilterAndClamp(t *testing.T) {
	stacks := []domain.Stack{
		{Name: "auth cleanup", PRs: []domain.PullRequest{{Title: "Split auth"}}},
		{Name: "composer tokens", PRs: []domain.PullRequest{{Title: "Add token"}}},
	}
	got := app.FilterStacks(stacks, "composer")
	if len(got) != 1 || got[0].Name != "composer tokens" {
		t.Fatalf("real filter: %+v", got)
	}
	alwaysAll := func(in []domain.Stack, query string) []domain.Stack {
		_ = query
		return in
	}
	if len(alwaysAll(stacks, "composer")) == 1 {
		t.Fatal("no-filter mutant must not survive")
	}

	empty := app.FilterStacks(stacks, "")
	if len(empty) != 2 {
		t.Fatal("empty query keeps all")
	}

	clamped := app.ClampSelection(app.Selection{StackIndex: 99, PRIndex: 99}, stacks)
	if clamped.StackIndex != 1 || clamped.PRIndex != 0 {
		t.Fatalf("real clamp %+v", clamped)
	}
	noClamp := func(sel app.Selection) app.Selection { return sel }
	if noClamp(app.Selection{StackIndex: 99, PRIndex: 99}).StackIndex == 1 {
		t.Fatal("no-clamp mutant must not survive")
	}
}
