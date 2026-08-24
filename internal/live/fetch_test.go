package live

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gsimone/dango-tui/internal/domain"
)

func ghOK(handlers map[string][]byte) runner {
	return func(args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		for prefix, raw := range handlers {
			if strings.HasPrefix(joined, prefix) {
				return raw, nil
			}
		}
		return nil, fmt.Errorf("unexpected gh %s", joined)
	}
}

func TestFetchWithGroupsChainFromGhJSON(t *testing.T) {
	run := ghOK(map[string][]byte{
		"repo view": []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`),
		"pr list": []byte(`[
				{"number":1,"title":"bottom","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","headRefOid":"aaa","author":{"login":"gm"},"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"APPROVED","latestReviews":[{"state":"APPROVED"}],"additions":1,"deletions":0,"changedFiles":1,"body":"","statusCheckRollup":[{"state":"SUCCESS"}]},
				{"number":2,"title":"top","url":"https://github.com/owner/demo/pull/2","headRefName":"b","baseRefName":"a","headRefOid":"bbb","author":{"login":"gm"},"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"","latestReviews":[],"additions":3,"deletions":1,"changedFiles":2,"body":"","statusCheckRollup":[{"state":"FAILURE"}]}
			]`),
		"api repos/": []byte(`[]`),
	})
	stacks, err := fetchWith(run, "owner/demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 1 || len(stacks[0].PRs) != 2 {
		t.Fatalf("want one chain of 2, got %+v", stacks)
	}
	if stacks[0].PRs[0].Number != 1 || stacks[0].PRs[1].Number != 2 {
		t.Fatalf("order %+v", numbers(stacks[0].PRs))
	}
	if stacks[0].PRs[1].CI.State != "failure" {
		t.Fatalf("ci %s", stacks[0].PRs[1].CI.State)
	}
	if stacks[0].PRs[0].Author != "gm" || stacks[0].PRs[0].AuthorColor != domain.LoginColor("gm") {
		t.Fatalf("author fallback: %+v", stacks[0].PRs[0])
	}
	if len(stacks[0].PRs[0].Labels) != 0 {
		t.Fatalf("no labels in this fixture: %+v", stacks[0].PRs[0].Labels)
	}
}

func TestFetchMapsLabelsAndAuthor(t *testing.T) {
	run := ghOK(map[string][]byte{
		"repo view": []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`),
		"pr list": []byte(`[
				{"number":1,"title":"bottom","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","headRefOid":"aaa","author":{"login":"gm","avatarUrl":"https://avatars.example/gm.png"},"labels":[{"name":"bug","color":"d73a4a"},{"name":"auth","color":"0e8a16"}],"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"APPROVED","latestReviews":[{"state":"APPROVED"}],"additions":1,"deletions":0,"changedFiles":1,"body":"","statusCheckRollup":[{"state":"SUCCESS"}]}
			]`),
		"api repos/": []byte(`[]`),
	})
	stacks, err := fetchWith(run, "owner/demo")
	if err != nil {
		t.Fatal(err)
	}
	pr := stacks[0].PRs[0]
	if len(pr.Labels) != 2 || pr.Labels[0].Name != "bug" || pr.Labels[0].Color != "#d73a4a" {
		t.Fatalf("labels %+v", pr.Labels)
	}
	if pr.Labels[1].Name != "auth" || pr.Labels[1].Color != "#0e8a16" {
		t.Fatalf("labels %+v", pr.Labels)
	}
	if pr.Author != "gm" || pr.AvatarURL != "https://avatars.example/gm.png" {
		t.Fatalf("author %+v", pr)
	}
	if pr.AuthorColor != domain.LoginColor("gm") {
		t.Fatalf("author ● is login-stable, got %q want %q", pr.AuthorColor, domain.LoginColor("gm"))
	}
	if domain.IsLowChromaHex(pr.AuthorColor) {
		t.Fatalf("author ● must stay chromatic: %s", pr.AuthorColor)
	}
}

func TestFetchWithRejectsEmptyRepo(t *testing.T) {
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		t.Fatal("must not call gh")
		return nil, nil
	}, "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchLookPathMissing(t *testing.T) {
	old := lookPath
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { lookPath = old })

	_, err := Fetch("owner/name")
	if !errors.Is(err, ErrGHMissing) {
		t.Fatalf("LookPath miss: %v", err)
	}
	if err.Error() != ErrGHMissing.Error() {
		t.Fatalf("sentence %q", err)
	}
}

func TestFetchWithMissingBinary(t *testing.T) {
	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls.Add(1)
		return nil, &exec.Error{Name: "gh", Err: exec.ErrNotFound}
	}, "owner/name")
	if calls.Load() == 0 {
		t.Fatal("injected run must be used")
	}
	if !errors.Is(err, ErrGHMissing) {
		t.Fatalf("missing binary: %v", err)
	}
	if err.Error() != "dango: gh CLI not found. Install https://cli.github.com and retry." {
		t.Fatalf("sentence %q", err)
	}
}

func TestPRListFieldsAreSlim(t *testing.T) {
	want := []string{
		"number", "title", "url", "headRefName", "baseRefName",
		"author", "labels", "isDraft", "state",
	}
	if strings.Join(prListFields, ",") != strings.Join(want, ",") {
		t.Fatalf("first list fields %v, want %v", prListFields, want)
	}
	joined := strings.Join(prListFields, ",")
	for _, banned := range []string{
		"body", "statusCheckRollup", "latestReviews", "reviews",
		"mergeStateStatus", "mergeable", "additions", "deletions",
		"changedFiles", "headRefOid",
	} {
		if strings.Contains(joined, banned) {
			t.Fatalf("first list must not request %s: %v", banned, prListFields)
		}
	}
}

func TestFetchDoesNotCallStacks(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	var jsonFields string
	run := func(args ...string) ([]byte, error) {
		mu.Lock()
		if len(args) > 0 {
			seen[args[0]] = true
		}
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "pr list") {
			for i, a := range args {
				if a == "--json" && i+1 < len(args) {
					jsonFields = args[i+1]
				}
			}
		}
		if args[0] == "api" {
			t.Fatal(" /stacks is not on the hot path")
		}
		mu.Unlock()
		return ghOK(map[string][]byte{
			"repo view": []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`),
			"pr list":   []byte(`[]`),
		})(args...)
	}
	if _, err := fetchWith(run, "owner/demo"); err != nil {
		t.Fatal(err)
	}
	if seen["api"] || seen["stacks"] {
		t.Fatalf("/stacks was called: %v", seen)
	}
	if !seen["repo"] || !seen["pr"] {
		t.Fatalf("need repo view + pr list, got %v", seen)
	}
	if strings.Contains(jsonFields, "body") || strings.Contains(jsonFields, "statusCheckRollup") {
		t.Fatalf("pr list still heavy: %s", jsonFields)
	}
}

func TestFetchSlimJSONStillGroups(t *testing.T) {
	run := ghOK(map[string][]byte{
		"repo view": []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`),
		"pr list": []byte(`[
			{"number":1,"title":"bottom","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","author":{"login":"gm"},"labels":[],"isDraft":false,"state":"OPEN"},
			{"number":2,"title":"top","url":"https://github.com/owner/demo/pull/2","headRefName":"b","baseRefName":"a","author":{"login":"gm"},"labels":[],"isDraft":false,"state":"OPEN"}
		]`),
	})
	stacks, err := fetchWith(run, "owner/demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 1 || len(stacks[0].PRs) != 2 {
		t.Fatalf("slim list must still chain, got %+v", stacks)
	}
}

func TestFetchRetries502ThenSucceeds(t *testing.T) {
	var slept []time.Duration
	oldSleep := sleep
	sleep = func(d time.Duration) { slept = append(slept, d) }
	t.Cleanup(func() { sleep = oldSleep })

	var listCalls atomic.Int32
	run := func(args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "repo view") {
			return []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`), nil
		}
		if strings.HasPrefix(joined, "pr list") {
			n := listCalls.Add(1)
			if n < 3 {
				return nil, fmt.Errorf("gh pr list: HTTP 502: Bad Gateway (https://api.github.com/graphql)")
			}
			return []byte(`[]`), nil
		}
		t.Fatalf("unexpected gh %s", joined)
		return nil, nil
	}
	if _, err := fetchWith(run, "owner/demo"); err != nil {
		t.Fatalf("502 should retry: %v", err)
	}
	if listCalls.Load() != 3 {
		t.Fatalf("list calls %d", listCalls.Load())
	}
	if len(slept) != 2 {
		t.Fatalf("backoff waits %v", slept)
	}
}

func TestFetch503RetriesThenErrors(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "repo view") {
			return []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`), nil
		}
		calls.Add(1)
		return nil, fmt.Errorf("HTTP 503: Service Unavailable")
	}, "owner/demo")
	if err == nil {
		t.Fatal("exhausted retries must fail")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("keep the 503: %v", err)
	}
	if calls.Load() != retryLimit {
		t.Fatalf("list 503 should stop after %d, got %d", retryLimit, calls.Load())
	}
}

func TestFetchAuthIsNotA502(t *testing.T) {
	sleep = func(time.Duration) { t.Fatal("auth must not backoff") }
	t.Cleanup(func() { sleep = time.Sleep })

	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls.Add(1)
		return nil, fmt.Errorf("gh pr list: HTTP 401: Bad credentials")
	}, "owner/demo")
	if !errors.Is(err, ErrGHAuth) {
		t.Fatalf("auth: %v", err)
	}
	if strings.Contains(err.Error(), "502") {
		t.Fatalf("must not report auth as 502: %v", err)
	}
	if !strings.Contains(err.Error(), "authentication or permission") {
		t.Fatalf("say auth plainly: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("auth must not retry, calls=%d", calls.Load())
	}
}

func TestFetchIsSerial(t *testing.T) {
	var current, max atomic.Int32
	run := func(args ...string) ([]byte, error) {
		n := current.Add(1)
		for {
			old := max.Load()
			if n <= old || max.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		current.Add(-1)
		return ghOK(map[string][]byte{
			"repo view": []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`),
			"pr list":   []byte(`[]`),
		})(args...)
	}
	if _, err := fetchWith(run, "owner/demo"); err != nil {
		t.Fatal(err)
	}
	if max.Load() != 1 {
		t.Fatalf("stampeded the API, max in flight %d", max.Load())
	}
}

func TestGHCommandInheritsEnv(t *testing.T) {
	cmd := ghCommand("pr", "list")
	if cmd.Env != nil {
		t.Fatalf("cmd.Env must stay nil so GH_TOKEN/GH_HOST inherit, got %v", cmd.Env)
	}
}

func TestRunGHNotFound(t *testing.T) {
	err := mapGHError(&exec.Error{Name: "gh", Err: exec.ErrNotFound}, "exec: \"gh\": executable file not found in $PATH", []string{"pr", "list"})
	if !errors.Is(err, ErrGHMissing) {
		t.Fatalf("runGH not-found: %v", err)
	}
	if strings.Contains(err.Error(), "executable file not found") {
		t.Fatalf("must not bury the exec error: %v", err)
	}

	t.Setenv("PATH", "")
	_, err = runGH("pr", "list")
	if !errors.Is(err, ErrGHMissing) {
		t.Fatalf("PATH-empty runGH: %v", err)
	}
}
