package live

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os/exec"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
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
	if stacks[0].PRs[0].Author != "gm" || stacks[0].PRs[0].AuthorColor != domain.LoginColor("gm") {
		t.Fatalf("author fallback: %+v", stacks[0].PRs[0])
	}
	if len(stacks[0].PRs[0].Labels) != 0 {
		t.Fatalf("no labels in this fixture: %+v", stacks[0].PRs[0].Labels)
	}
}

func TestFetchMapsLabelsAndAuthor(t *testing.T) {
	pngBytes := solidPNG(t, 200, 40, 40)
	old := getURL
	getURL = func(raw string) ([]byte, error) {
		if raw != "https://avatars.example/gm.png" {
			t.Fatalf("avatar url %q", raw)
		}
		return pngBytes, nil
	}
	t.Cleanup(func() { getURL = old })

	run := func(args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "repo view"):
			return []byte(`{"nameWithOwner":"owner/demo","defaultBranchRef":{"name":"main"}}`), nil
		case strings.HasPrefix(joined, "pr list"):
			return []byte(`[
				{"number":1,"title":"bottom","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","headRefOid":"aaa","author":{"login":"gm","avatarUrl":"https://avatars.example/gm.png"},"labels":[{"name":"bug","color":"d73a4a"},{"name":"auth","color":"0e8a16"}],"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"APPROVED","latestReviews":[{"state":"APPROVED"}],"additions":1,"deletions":0,"changedFiles":1,"body":"","statusCheckRollup":[{"state":"SUCCESS"}]}
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
	if pr.AuthorColor != "#c82828" {
		t.Fatalf("sampled avatar color %q", pr.AuthorColor)
	}
}

func TestAvatarFetchFailureFallsBackToLogin(t *testing.T) {
	old := getURL
	getURL = func(string) ([]byte, error) { return nil, errors.New("nope") }
	t.Cleanup(func() { getURL = old })
	if got := resolveAuthorColor("gm", "https://avatars.example/x.png"); got != domain.LoginColor("gm") {
		t.Fatalf("fallback %q", got)
	}
	if got := resolveAuthorColor("gm", ""); got != domain.LoginColor("gm") {
		t.Fatalf("empty url %q", got)
	}
}

func solidPNG(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
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
