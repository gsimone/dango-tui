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
		t.Fatal("two-layer stacks get a distinct clause in the pane")
	}
	if untitled[0].Name == "base" || untitled[0].Description == "base" {
		t.Fatalf("apply must not echo the gh title: %+v", untitled[0])
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
	named.PRs[0].Body = "<!-- CURSOR_AGENT_PR_BODY_BEGIN --> raw body dump"
	kept := summary.Run(summary.Job{
		Provider: summary.ParseProvider("local"),
		ID:       "stack-1",
		Stack:    named,
	})
	if kept.Description == "Already summarized by a model." || strings.Contains(kept.Description, "CURSOR_AGENT") || strings.Contains(kept.Description, "raw body") {
		t.Fatalf("local does not paste body or stored dump: %q", kept.Description)
	}
	if strings.HasPrefix(kept.Description, "Covers ") {
		t.Fatalf("do not invent a Covers wrapper: %q", kept.Description)
	}
}

func TestRunDoesNotEchoGhTitle(t *testing.T) {
	stack := domain.Stack{
		ID:   "stack-1",
		Name: "LEV-182: Bound hosts to the session",
		PRs:  []domain.PullRequest{{Number: 182, Title: "LEV-182: Bound hosts to the session"}},
	}
	gh := "LEV-182: Bound hosts to the session"
	res := summary.Run(summary.Job{
		Provider: summary.ParseProvider("local"),
		ID:       "stack-1",
		Stack:    stack,
	})
	if res.Title == gh || strings.EqualFold(res.Title, gh) {
		t.Fatalf("title echoed gh name: %q", res.Title)
	}
	if res.Description == gh || strings.EqualFold(res.Description, gh) {
		t.Fatalf("description must not paste the gh title, got %q", res.Description)
	}
	if strings.HasPrefix(res.Description, "Covers ") {
		t.Fatalf("do not invent a Covers wrapper: %q", res.Description)
	}

	withBody := stack
	withBody.PRs[0].Body = "Pin each bound host to the worker that opened the session.\n\nManaged by Graphite.\n- #182\n"
	bodied := summary.Run(summary.Job{
		Provider: summary.ParseProvider("demo"),
		ID:       "stack-1",
		Stack:    withBody,
	})
	if strings.Contains(bodied.Description, "Pin each bound host") {
		t.Fatalf("must not paste pr.Body: %q", bodied.Description)
	}
	if strings.HasPrefix(bodied.Description, "Covers ") {
		t.Fatalf("do not invent a Covers wrapper: %q", bodied.Description)
	}
}

func TestDescribeIgnoresAgentPRBody(t *testing.T) {
	raw := "<!-- CURSOR_AGENT_PR_BODY_BEGIN -->\nLinear: [LEV-182](https://linear.app/leva/issue/LEV-182) Pin each bound host.\n<!-- CURSOR_AGENT_PR_BODY_END -->"
	stack := domain.Stack{
		Name: "LEV-182: Bound hosts to the session",
		PRs: []domain.PullRequest{{
			Title: "LEV-182: Bound hosts to the session",
			Body:  raw,
		}},
	}
	got := summary.Describe(stack)
	if strings.Contains(got, "CURSOR_AGENT") || strings.Contains(got, "<!--") || strings.Contains(got, "-->") {
		t.Fatalf("agent comment leaked: %q", got)
	}
	if strings.Contains(got, raw) || strings.Contains(got, "linear.app") || strings.Contains(got, "Pin each bound host") {
		t.Fatalf("raw body leaked: %q", got)
	}
	if strings.HasPrefix(got, "Covers ") {
		t.Fatalf("do not invent a Covers wrapper: %q", got)
	}
	if got != "" && (got == stack.PRs[0].Title || strings.EqualFold(got, stack.PRs[0].Title)) {
		t.Fatalf("must not be GhTitle: %q", got)
	}
}

func TestDescribeSkipsStub(t *testing.T) {
	stack := domain.Stack{
		Description: "A deterministic fixture stack",
		PRs:         []domain.PullRequest{{Title: "Split auth scope"}},
	}
	got := summary.Describe(stack)
	if got == "A deterministic fixture stack" {
		t.Fatalf("stub must not be shown: %q", got)
	}
	if got != "" && strings.EqualFold(got, "Split auth scope") {
		t.Fatalf("single-layer gh title is not a description: %q", got)
	}
	if strings.HasPrefix(got, "Covers ") {
		t.Fatalf("do not invent a Covers wrapper: %q", got)
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
