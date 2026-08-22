package live

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestFetchWithGroupsChainFromGhJSON(t *testing.T) {
	run := func(args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "repo view"):
			return []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`), nil
		case strings.HasPrefix(joined, "pr list"):
			return []byte(`[
				{"number":1,"title":"bottom","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","headRefOid":"aaa","author":{"login":"gm"},"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"APPROVED","latestReviews":[{"state":"APPROVED"}],"additions":1,"deletions":0,"changedFiles":1,"body":"","statusCheckRollup":[{"state":"SUCCESS"}]},
				{"number":2,"title":"top","url":"https://github.com/owner/demo/pull/2","headRefName":"b","baseRefName":"a","headRefOid":"bbb","author":{"login":"gm"},"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"","latestReviews":[],"additions":3,"deletions":1,"changedFiles":2,"body":"","statusCheckRollup":[{"state":"FAILURE"}]}
			]`), nil
		case strings.HasPrefix(joined, "api repos/"):
			return []byte(`[]`), nil
		default:
			t.Fatalf("unexpected gh %s", joined)
			return nil, nil
		}
	}
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
	calls := 0
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls++
		return nil, &exec.Error{Name: "gh", Err: exec.ErrNotFound}
	}, "owner/name")
	if calls == 0 {
		t.Fatal("injected run must be used")
	}
	if !errors.Is(err, ErrGHMissing) {
		t.Fatalf("missing binary: %v", err)
	}
	if err.Error() != "dango: gh CLI not found. Install https://cli.github.com and retry." {
		t.Fatalf("sentence %q", err)
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
