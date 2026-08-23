package summary_test

import (
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/summary"
)

func TestFromLayersWritesOneClause(t *testing.T) {
	var s summary.FromLayers
	mixed := data.StoryByID("mixed")
	want := map[string][]string{
		"auth cleanup":    {"split auth scope", "keep service identity", "implicit session fallback"},
		"composer tokens": {"token catalogue", "entity fields", "token migration"},
		"sync rewrite":    {"syncing work", "duplicate invalidation"},
	}
	for _, stack := range mixed.Stacks {
		needles, ok := want[stack.Name]
		if !ok {
			t.Fatalf("unexpected mixed stack %q", stack.Name)
		}
		got := s.Summarize(stack)
		if strings.Contains(got, "\n") {
			t.Fatalf("%s must be one line: %q", stack.Name, got)
		}
		if strings.Contains(got, "A deterministic fixture stack") {
			t.Fatalf("%s used the stub: %s", stack.Name, got)
		}
		lower := strings.ToLower(got)
		for _, needle := range needles {
			if !strings.Contains(lower, needle) {
				t.Fatalf("%s clause missing %q:\n%s", stack.Name, needle, got)
			}
		}
	}
}

func TestFromDescriptionPassthrough(t *testing.T) {
	var s summary.FromDescription
	stack := domain.Stack{Name: "auth cleanup", Description: "Simplify authentication boundaries"}
	if got := s.Summarize(stack); got != "Simplify authentication boundaries" {
		t.Fatalf("passthrough: %q", got)
	}
}

func TestPreferDescriptionSkipsStub(t *testing.T) {
	p := summary.PreferDescription{Fallback: summary.FromLayers{}}
	stub := data.StackFixture(data.StackFixtureInput{
		Name: "auth cleanup",
		PRs: []domain.PullRequest{
			data.PRFixture(data.PullRequestFixtureInput{Number: 184, Title: "Split auth scope from session checks", State: domain.StateMerged}),
		},
	})
	got := p.Summarize(stub)
	if got == "A deterministic fixture stack" || got == "" {
		t.Fatalf("stub should fall through: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "split auth scope") {
		t.Fatalf("fallback clause: %q", got)
	}

	named := stub
	named.Description = "Already summarized by a model."
	if got := p.Summarize(named); got != "Already summarized by a model." {
		t.Fatalf("prefer set description: %q", got)
	}
}

func TestChooseThreadsProviderAndStaysLocal(t *testing.T) {
	p := summary.ParseProvider("codex@luna.medium")
	if p.Name != "codex" || p.Model != "luna.medium" {
		t.Fatalf("parse %+v", p)
	}
	got := summary.Choose(p)
	if got.Provider != p {
		t.Fatalf("choose must keep provider, got %+v", got.Provider)
	}
	stack := data.StoryByID("mixed").Stacks[0]
	local := summary.Local().Summarize(stack)
	if got.Summarize(stack) != local {
		t.Fatalf("provider with no network summarizer uses Local; want %q, got %q", local, got.Summarize(stack))
	}
	empty := summary.Choose(summary.Provider{})
	if empty.Summarize(stack) != "" {
		t.Fatalf("missing provider must not invent a title, got %q", empty.Summarize(stack))
	}
	untitled := []domain.Stack{{
		PRs: []domain.PullRequest{{Number: 1, Title: "base"}, {Number: 2, Title: "head"}},
	}}
	summary.Apply(untitled, empty)
	if untitled[0].Name != "" || untitled[0].Summary != "" {
		t.Fatalf("apply without provider filled a title: %+v", untitled[0])
	}
	summary.Apply(untitled, got)
	if untitled[0].Name == "" {
		t.Fatal("provider should write the stack title")
	}
	if untitled[0].Description == "" {
		t.Fatal("provider should write inspector description")
	}
}

func TestRunWritesTitleAndDescription(t *testing.T) {
	stack := domain.Stack{
		ID: "stack-1",
		PRs: []domain.PullRequest{
			{Title: "Alpha layer"},
			{Title: "Beta layer"},
		},
	}
	empty := summary.Run(summary.Job{ID: "stack-1", Stack: stack})
	if empty.ID != "stack-1" {
		t.Fatalf("id %q", empty.ID)
	}
	if empty.Title != "" || empty.Description != "" {
		t.Fatalf("missing provider must not invent a title: %+v", empty)
	}

	res := summary.Run(summary.Job{
		Provider: summary.ParseProvider("codex@luna.medium"),
		ID:       "stack-1",
		Stack:    stack,
	})
	if res.ID != "stack-1" {
		t.Fatalf("id %q", res.ID)
	}
	if res.Title == "" || res.Title == "Alpha layer" {
		t.Fatalf("provider must write a generated title, got %q", res.Title)
	}
	if !strings.Contains(strings.ToLower(res.Title), "alpha layer") || !strings.Contains(strings.ToLower(res.Title), "beta layer") {
		t.Fatalf("title should come from the layers: %q", res.Title)
	}
	if strings.Contains(res.Title, "\n") {
		t.Fatalf("title must be one line: %q", res.Title)
	}
	if res.Description == "" {
		t.Fatalf("provider must write a description")
	}
	if strings.Contains(res.Description, "\n") {
		t.Fatalf("description must be one line: %q", res.Description)
	}
	if !strings.Contains(strings.ToLower(res.Description), "alpha") || !strings.Contains(strings.ToLower(res.Description), "beta") {
		t.Fatalf("description should cover the layers: %q", res.Description)
	}

	named := stack
	named.Description = "Already summarized by a model."
	kept := summary.Run(summary.Job{
		Provider: summary.ParseProvider("local"),
		ID:       "stack-1",
		Stack:    named,
	})
	if kept.Description != "Already summarized by a model." {
		t.Fatalf("real description should win: %q", kept.Description)
	}
}

func TestDescribeSkipsStub(t *testing.T) {
	stack := domain.Stack{
		Description: "A deterministic fixture stack",
		PRs:         []domain.PullRequest{{Title: "Split auth scope"}},
	}
	got := summary.Describe(stack)
	if got == "" || got == "A deterministic fixture stack" {
		t.Fatalf("stub should fall through: %q", got)
	}
	if !strings.Contains(strings.ToLower(got), "split auth scope") {
		t.Fatalf("layer clause: %q", got)
	}
}

func TestLoadStoresSummariesOnFixtureStacks(t *testing.T) {
	stack := data.StoryByID("mixed").Stacks[0]
	if stack.Summary == "" {
		t.Fatal("load should store a summary on the stack")
	}
	if strings.Contains(stack.Summary, "\n") {
		t.Fatalf("stored summary must be one line: %q", stack.Summary)
	}
	if strings.Contains(stack.Summary, "A deterministic fixture stack") {
		t.Fatalf("stored stub: %s", stack.Summary)
	}
	if !strings.Contains(strings.ToLower(stack.Summary), "split auth scope") {
		t.Fatalf("stored summary: %s", stack.Summary)
	}
}
