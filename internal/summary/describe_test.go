package summary

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func sampleStack() domain.Stack {
	return domain.Stack{
		ID: "s",
		PRs: []domain.PullRequest{
			{Number: 182, Title: "LEV-182: Bound hosts to the session"},
			{Number: 183, Title: "Pin each host to the worker"},
		},
	}
}

func TestRunWithoutDescribeUsesLocal(t *testing.T) {
	stack := sampleStack()
	called := false
	old := runDescribe
	runDescribe = func(context.Context, []string, []byte) (string, error) {
		called = true
		return "should not run", nil
	}
	t.Cleanup(func() { runDescribe = old })

	res := Run(Job{ID: "s", Stack: stack})
	if called {
		t.Fatal("no describe config must not spawn a script")
	}
	if res.Title != "" {
		t.Fatalf("no provider must not retitle: %+v", res)
	}
	if res.Description != Describe(stack) {
		t.Fatalf("local Describe() fallback: %q", res.Description)
	}
}

func TestRunPrefersDescribeScript(t *testing.T) {
	stack := sampleStack()
	var sawArgv []string
	var sawStdin []byte
	old := runDescribe
	runDescribe = func(ctx context.Context, argv []string, stdin []byte) (string, error) {
		if ctx == nil {
			t.Fatal("runner must get a timeout context")
		}
		sawArgv = append([]string(nil), argv...)
		sawStdin = append([]byte(nil), stdin...)
		return "hosts stay pinned so undo cannot widen scope", nil
	}
	t.Cleanup(func() { runDescribe = old })

	res := Run(Job{ID: "s", Describe: "scripts/dango-describe", Stack: stack})
	if res.Title != "" {
		t.Fatalf("no provider must not retitle: %+v", res)
	}
	if res.Description != "hosts stay pinned so undo cannot widen scope" {
		t.Fatalf("script description: %q", res.Description)
	}
	if res.Description == Describe(stack) {
		t.Fatal("product sentence is the script, not Describe()")
	}
	if len(sawArgv) != 1 || sawArgv[0] != "scripts/dango-describe" {
		t.Fatalf("configured script argv: %v", sawArgv)
	}
	joined := strings.Join(sawArgv, " ")
	for _, bad := range []string{"OPENAI_API_KEY", "DANGO_OPENAI_API_KEY", "chat/completions", "codex"} {
		if strings.Contains(joined, bad) {
			t.Fatalf("binary must not assume codex/key: %s", joined)
		}
	}
	var payload describeInput
	if err := json.Unmarshal(sawStdin, &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Titles) != 2 || payload.Titles[0] != "LEV-182: Bound hosts to the session" {
		t.Fatalf("stdin titles: %+v", payload)
	}
	if strings.Contains(string(sawStdin), "CURSOR_AGENT") || strings.Contains(string(sawStdin), "secret") {
		t.Fatalf("stdin leaked body: %s", sawStdin)
	}
}

func TestRunDescribeOmitsBody(t *testing.T) {
	stack := sampleStack()
	stack.PRs[0].Body = "<!-- CURSOR_AGENT_PR_BODY_BEGIN -->secret body"
	old := runDescribe
	runDescribe = func(_ context.Context, _ []string, stdin []byte) (string, error) {
		if strings.Contains(string(stdin), "secret body") || strings.Contains(string(stdin), "CURSOR_AGENT") {
			t.Fatalf("body leaked: %s", stdin)
		}
		return "hosts stay pinned", nil
	}
	t.Cleanup(func() { runDescribe = old })
	res := Run(Job{Describe: "./describe", Stack: stack})
	if res.Description != "hosts stay pinned" {
		t.Fatalf("got %q", res.Description)
	}
}

func TestRunFallsBackToDescribe(t *testing.T) {
	stack := sampleStack()
	old := runDescribe
	t.Cleanup(func() { runDescribe = old })

	runDescribe = func(context.Context, []string, []byte) (string, error) {
		return "", errors.New("exit 1")
	}
	res := Run(Job{ID: "s", Describe: "scripts/dango-describe", Stack: stack})
	if res.Title != "" || res.Description != Describe(stack) {
		t.Fatalf("non-zero uses Describe(): %+v", res)
	}

	runDescribe = func(context.Context, []string, []byte) (string, error) {
		return "", context.DeadlineExceeded
	}
	timed := Run(Job{ID: "s", Describe: "scripts/dango-describe", Stack: stack})
	if timed.Description != Describe(stack) {
		t.Fatalf("timeout uses Describe(): %q", timed.Description)
	}

	runDescribe = func(context.Context, []string, []byte) (string, error) { return "", nil }
	empty := Run(Job{ID: "s", Describe: "scripts/dango-describe", Stack: stack})
	if empty.Description != Describe(stack) {
		t.Fatalf("empty stdout uses Describe(): %q", empty.Description)
	}
}

func TestRunMushUsesDescribe(t *testing.T) {
	stack := sampleStack()
	old := runDescribe
	t.Cleanup(func() { runDescribe = old })
	for _, mush := range []string{
		"",
		"Covers the bound hosts",
		"<!-- CURSOR_AGENT --> dump",
		Describe(stack),
		ghName(stack),
	} {
		runDescribe = func(context.Context, []string, []byte) (string, error) { return mush, nil }
		res := Run(Job{ID: "s", Describe: "scripts/dango-describe", Stack: stack})
		if res.Description != Describe(stack) {
			t.Fatalf("mush %q must fall back, got %q", mush, res.Description)
		}
		if res.Title != "" {
			t.Fatalf("mush must not retitle: %+v", res)
		}
	}
}

func TestRunProviderTitleKeepsScriptDescription(t *testing.T) {
	stack := sampleStack()
	old := runDescribe
	runDescribe = func(context.Context, []string, []byte) (string, error) {
		return "hosts stay pinned so undo cannot widen scope", nil
	}
	t.Cleanup(func() { runDescribe = old })

	res := Run(Job{
		Provider: ParseProvider("codex@luna.medium"),
		Describe: "scripts/dango-describe",
		ID:       "s",
		Stack:    stack,
	})
	if res.Title == "" || res.Title == stack.PRs[0].Title {
		t.Fatalf("provider may still swap a title: %+v", res)
	}
	if res.Description != "hosts stay pinned so undo cannot widen scope" {
		t.Fatalf("script still writes the pane: %q", res.Description)
	}
}

func TestCleanDescribeRejectsMush(t *testing.T) {
	stack := sampleStack()
	if got := cleanDescribe("Covers the bound hosts", stack); got != "" {
		t.Fatalf("Covers: %q", got)
	}
	if got := cleanDescribe("x CURSOR_AGENT y", stack); got != "" {
		t.Fatalf("CURSOR_AGENT: %q", got)
	}
	if got := cleanDescribe(ghName(stack), stack); got != "" {
		t.Fatalf("title again: %q", got)
	}
	if got := cleanDescribe(Describe(stack), stack); got != "" {
		t.Fatalf("joined titles: %q", got)
	}
	got := cleanDescribe("Hosts stay pinned so undo cannot widen scope.", stack)
	if got != "Hosts stay pinned so undo cannot widen scope." {
		t.Fatalf("real sentence: %q", got)
	}
}

func TestDescribeArgvSplitsCommand(t *testing.T) {
	got := describeArgv("scripts/dango-describe --flag")
	if len(got) != 2 || got[0] != "scripts/dango-describe" || got[1] != "--flag" {
		t.Fatalf("%v", got)
	}
	if len(describeArgv("")) != 0 {
		t.Fatal("empty command is no script")
	}
}

func TestExecDescribeNeverRunsLiveInUnitTests(t *testing.T) {
	got, err := execDescribe(context.Background(), []string{"scripts/dango-describe"}, []byte(`{"titles":["a"]}`))
	if got != "" || !errors.Is(err, errNoDescribe) {
		t.Fatalf("unit tests must not spawn a describe script: %q %v", got, err)
	}
}

func TestExampleDescribeScriptPinsLuna(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "scripts", "dango-describe"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "codex exec") {
		t.Fatal("example script calls the Codex CLI")
	}
	if !strings.Contains(s, "-m") || !strings.Contains(s, "gpt-5.6-luna") {
		t.Fatal("example must pass gpt-5.6 luna, not the CLI default")
	}
	if strings.Contains(s, "OPENAI_API_KEY") || strings.Contains(s, "chat/completions") {
		t.Fatal("example is the CLI, not an API key")
	}
}
