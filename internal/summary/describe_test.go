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

func TestRunWithoutDescribeLeavesEmpty(t *testing.T) {
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
		t.Fatal("no describe key must not spawn a script")
	}
	if res.Title != "" {
		t.Fatalf("no provider must not retitle: %+v", res)
	}
	if res.Description != "" {
		t.Fatalf("unset describe leaves the pane empty, got %q (Describe() is not the product)", res.Description)
	}
	if res.Description == Describe(stack) && Describe(stack) != "" {
		t.Fatal("Describe() must not run as the product sentence")
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

	res := Run(Job{ID: "s", Describe: "bin/describe-stack", Stack: stack})
	if res.Title != "" {
		t.Fatalf("no provider must not retitle: %+v", res)
	}
	if res.Description != "hosts stay pinned so undo cannot widen scope" {
		t.Fatalf("script description: %q", res.Description)
	}
	if res.Description == Describe(stack) {
		t.Fatal("product sentence is the script, not Describe()")
	}
	if len(sawArgv) != 1 || sawArgv[0] != "bin/describe-stack" {
		t.Fatalf("configured script argv: %v", sawArgv)
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

func TestRunScriptFailureLeavesEmpty(t *testing.T) {
	stack := sampleStack()
	old := runDescribe
	t.Cleanup(func() { runDescribe = old })

	runDescribe = func(context.Context, []string, []byte) (string, error) {
		return "", errors.New("exit 1")
	}
	res := Run(Job{ID: "s", Describe: "bin/describe-stack", Stack: stack})
	if res.Title != "" || res.Description != "" {
		t.Fatalf("non-zero leaves the pane empty: %+v", res)
	}
	if strings.Contains(res.Description, "exit 1") {
		t.Fatal("never paint stderr / errors into the inspector")
	}

	runDescribe = func(context.Context, []string, []byte) (string, error) {
		return "", context.DeadlineExceeded
	}
	timed := Run(Job{ID: "s", Describe: "bin/describe-stack", Stack: stack})
	if timed.Description != "" {
		t.Fatalf("timeout leaves empty: %q", timed.Description)
	}

	runDescribe = func(context.Context, []string, []byte) (string, error) {
		return "fatal: command failed\n", errors.New("exit 2")
	}
	errOut := Run(Job{ID: "s", Describe: "bin/describe-stack", Stack: stack})
	if errOut.Description != "" {
		t.Fatalf("error stdout must not land: %q", errOut.Description)
	}
}

func TestRunMushLeavesEmpty(t *testing.T) {
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
		res := Run(Job{ID: "s", Describe: "bin/describe-stack", Stack: stack})
		if res.Description != "" {
			t.Fatalf("mush %q must leave the pane empty, got %q", mush, res.Description)
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
		Provider: ParseProvider("name@model"),
		Describe: "bin/describe-stack",
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
	got := describeArgv("bin/describe-stack --flag")
	if len(got) != 2 || got[0] != "bin/describe-stack" || got[1] != "--flag" {
		t.Fatalf("%v", got)
	}
	if len(describeArgv("")) != 0 {
		t.Fatal("empty command is no script")
	}
}

func TestExecDescribeNeverRunsLiveInUnitTests(t *testing.T) {
	got, err := execDescribe(context.Background(), []string{"bin/describe-stack"}, []byte(`{"titles":["a"]}`))
	if got != "" || !errors.Is(err, errNoDescribe) {
		t.Fatalf("unit tests must not spawn a describe script: %q %v", got, err)
	}
}

func TestProductGoHasNoCodexString(t *testing.T) {
	root := filepath.Join("..", "..")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			base := info.Name()
			if base == ".git" || base == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(raw), "codex") || strings.Contains(string(raw), "CODEX") {
			t.Errorf("product source must not contain that CLI name: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
