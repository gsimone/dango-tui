package app_test

import (
	"testing"

	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/data"
)

func TestFilterStacksKeepsTopology(t *testing.T) {
	stacks := data.StoryByID("mixed").Stacks
	filtered := app.FilterStacks(stacks, "composer")
	if len(filtered) != 1 || filtered[0].Name != "composer tokens" {
		t.Fatalf("filter composer: %+v", filtered)
	}
	if app.FilterStacks(stacks, "")[0].Name != "auth cleanup" {
		t.Fatal("empty query should keep fixture order")
	}
	if got := app.FilterStacks(stacks, "no-such-stack"); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}

func TestMoveSelectionAndClamp(t *testing.T) {
	stacks := data.StoryByID("mixed").Stacks
	sel := app.Selection{StackIndex: 0, PRIndex: 0}

	right := app.MoveSelection(sel, stacks, app.DirRight)
	if right.PRIndex != 1 {
		t.Fatalf("right: %+v", right)
	}
	end := app.MoveSelection(sel, stacks, app.DirEnd)
	if end.StackIndex != 2 {
		t.Fatalf("end of list: %+v", end)
	}
	down := app.MoveSelection(sel, stacks, app.DirDown)
	if down.StackIndex != 1 || down.PRIndex != 0 {
		t.Fatalf("down: %+v", down)
	}
	home := app.MoveSelection(end, stacks, app.DirHome)
	if home.StackIndex != 0 {
		t.Fatalf("home of list: %+v", home)
	}
	clamped := app.ClampSelection(app.Selection{StackIndex: 99, PRIndex: 99}, stacks)
	if clamped.StackIndex != 2 || clamped.PRIndex != 1 {
		t.Fatalf("clamp: %+v", clamped)
	}
	empty := app.ClampSelection(app.Selection{StackIndex: 3, PRIndex: 2}, nil)
	if empty != (app.Selection{}) {
		t.Fatalf("empty clamp: %+v", empty)
	}
}
