package summary_test

import (
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/summary"
)

func TestMutantTitleDoesNotEchoGh(t *testing.T) {
	stack := domain.Stack{
		Name: "LEV-182: Bound hosts to the session",
		PRs:  []domain.PullRequest{{Title: "LEV-182: Bound hosts to the session"}},
	}
	got := summary.Title(stack)
	if got == "" || got == stack.PRs[0].Title {
		t.Fatalf("real title strips the ticket, got %q", got)
	}
	echo := func(s domain.Stack) string { return s.PRs[0].Title }
	if echo(stack) != stack.PRs[0].Title {
		t.Fatal("echo mutant setup")
	}
	if echo(stack) == got {
		t.Fatal("echo-gh mutant must not survive")
	}

	desc := summary.Describe(stack)
	if desc == stack.PRs[0].Title || strings.HasPrefix(desc, "Covers ") {
		t.Fatalf("real describe %q", desc)
	}
	covers := func(s domain.Stack) string { return "Covers " + s.PRs[0].Title }
	if !strings.HasPrefix(covers(stack), "Covers ") {
		t.Fatal("covers mutant setup")
	}
	if covers(stack) == desc {
		t.Fatal("Covers-wrapper mutant must not survive")
	}
}

func TestMutantRunNeedsProvider(t *testing.T) {
	stack := domain.Stack{ID: "s", PRs: []domain.PullRequest{{Title: "Alpha"}, {Title: "Beta"}}}
	empty := summary.Run(summary.Job{Stack: stack, ID: "s"})
	if empty.Title != "" {
		t.Fatalf("missing provider must not invent a title: %+v", empty)
	}
	if empty.Description == "" {
		t.Fatal("missing provider still writes Describe()")
	}
	alwaysTitle := func(job summary.Job) summary.Result {
		return summary.Result{ID: job.ID, Title: summary.Title(job.Stack), Description: summary.Describe(job.Stack)}
	}
	if alwaysTitle(summary.Job{Stack: stack, ID: "s"}).Title == "" {
		t.Fatal("always-title mutant must invent a title")
	}
	if alwaysTitle(summary.Job{Stack: stack, ID: "s"}).Title == empty.Title {
		t.Fatal("always-title mutant must not survive")
	}
}

func TestMutantRunKeepsDescribeAsFallback(t *testing.T) {
	stack := domain.Stack{ID: "s", PRs: []domain.PullRequest{{Title: "Alpha"}, {Title: "Beta"}}}
	res := summary.Run(summary.Job{Stack: stack, ID: "s"})
	if res.Title != "" {
		t.Fatalf("no provider must not retitle: %+v", res)
	}
	if res.Description != summary.Describe(stack) {
		t.Fatalf("no key uses Describe(), not a model sentence: got %q want %q", res.Description, summary.Describe(stack))
	}
	alwaysLocal := func(job summary.Job) summary.Result {
		return summary.Result{ID: job.ID, Description: summary.Describe(job.Stack)}
	}
	luna := "luna wrote the pane"
	if alwaysLocal(summary.Job{Stack: stack, ID: "s"}).Description == luna {
		t.Fatal("always-local mutant would ignore luna")
	}
	if res.Description == luna {
		t.Fatal("failed/missing luna must not invent a model sentence")
	}
}

func TestMutantParseProvider(t *testing.T) {
	p := summary.ParseProvider("codex@luna.medium")
	if p.Name != "codex" || p.Model != "luna.medium" {
		t.Fatalf("real parse %+v", p)
	}
	noSplit := func(raw string) summary.Provider {
		return summary.Provider{Raw: raw, Name: raw}
	}
	if noSplit("codex@luna.medium").Model == "luna.medium" {
		t.Fatal("no-split mutant must not survive")
	}
}
