package live

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

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

func TestPRListFieldsSkipReviews(t *testing.T) {
	hasLatest := false
	for _, field := range prListFields {
		if field == "reviews" {
			t.Fatalf("full review history is dead weight on the live fetch: %v", prListFields)
		}
		if field == "latestReviews" {
			hasLatest = true
		}
	}
	if !hasLatest {
		t.Fatal("keep latestReviews for approval counts")
	}
}

func TestFetchIssuesRepoListAndStacks(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]bool{}
	run := func(args ...string) ([]byte, error) {
		mu.Lock()
		if len(args) > 0 {
			seen[args[0]] = true
		}
		mu.Unlock()
		return ghOK(map[string][]byte{
			"repo view":  []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`),
			"pr list":    []byte(`[]`),
			"api repos/": []byte(`[]`),
		})(args...)
	}
	if _, err := fetchWith(run, "owner/demo"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"repo", "pr", "api"} {
		if !seen[want] {
			t.Fatalf("missing gh %s call: %v", want, seen)
		}
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
