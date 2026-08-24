package live

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

func flagValue(args []string, name string) string {
	for i, a := range args {
		if a == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestFetchUsesPRListNotREST(t *testing.T) {
	want := prListArgs("archetype-labs/app")
	var calls [][]string
	run := func(args ...string) ([]byte, error) {
		cp := append([]string(nil), args...)
		calls = append(calls, cp)
		joined := strings.Join(args, " ")
		if args[0] == "api" || strings.Contains(joined, "/pulls") {
			t.Fatalf("must not call gh api …/pulls: %v", args)
		}
		if args[0] == "repo" || strings.Contains(joined, "repo view") {
			t.Fatalf("must not call gh repo view: %v", args)
		}
		if strings.Contains(joined, "stacks") {
			t.Fatalf("must not call /stacks: %v", args)
		}
		if len(args) != len(want) {
			t.Fatalf("exact argv length %d, want %d: %v", len(args), len(want), args)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("argv[%d]=%q, want %q (full %v)", i, args[i], want[i], args)
			}
		}
		return []byte(`[]`), nil
	}
	if _, err := fetchWith(run, "archetype-labs/app"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("open is one pr list, got %v", calls)
	}
	if FormatGHArgv(LastGHArgv) != FormatGHArgv(want) {
		t.Fatalf("logged argv %q, want %q", FormatGHArgv(LastGHArgv), FormatGHArgv(want))
	}
}

func TestFetchErrorIncludesExactArgv(t *testing.T) {
	want := FormatGHArgv(prListArgs("archetype-labs/app"))
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		return nil, fmt.Errorf("HTTP 502: Bad Gateway")
	}, "archetype-labs/app")
	if err == nil {
		t.Fatal("expected 502")
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error must print exact argv %q, got %v", want, err)
	}
	if !strings.Contains(err.Error(), "--json") || !strings.Contains(err.Error(), "isDraft") {
		t.Fatalf("do not truncate argv: %v", err)
	}
	if !strings.Contains(err.Error(), "mergeable") || !strings.Contains(err.Error(), "reviewDecision") || !strings.Contains(err.Error(), "mergeStateStatus") {
		t.Fatalf("error argv must keep display-state fields: %v", err)
	}
	if strings.Contains(err.Error(), "statusCheckRollup") || strings.Contains(err.Error(), "body") {
		t.Fatalf("error argv must stay cheap: %v", err)
	}
	if strings.Contains(err.Error(), "/pulls") || strings.Contains(err.Error(), "repo view") {
		t.Fatalf("must stay on pr list: %v", err)
	}
}

func TestPRListFieldsAreSlim(t *testing.T) {
	want := []string{
		"number", "title", "url", "headRefName", "baseRefName",
		"author", "labels", "isDraft", "state",
		"mergeable", "reviewDecision", "mergeStateStatus",
	}
	if strings.Join(prListFields, ",") != strings.Join(want, ",") {
		t.Fatalf("first list fields %v, want %v", prListFields, want)
	}
	joined := strings.Join(prListFields, ",")
	// gh pr list --json has no avatarUrl field (author is login/id/name).
	// Avatar bytes come from author.avatarUrl when present, else github.com/{login}.png.
	if strings.Contains(joined, "avatarUrl") {
		t.Fatalf("gh pr list --json rejects avatarUrl: %v", prListFields)
	}
	for _, banned := range []string{
		"body", "statusCheckRollup", "latestReviews", "reviews",
		"additions", "deletions", "changedFiles", "headRefOid",
	} {
		if strings.Contains(joined, banned) {
			t.Fatalf("first list must not request %s: %v", banned, prListFields)
		}
	}
}

func TestFetchMapsStateColorTokens(t *testing.T) {
	type row struct {
		name  string
		extra string
		token string
		ci    domain.CIState
	}
	cases := []row{
		{name: "draft", extra: `"isDraft":true,"state":"OPEN"`, token: "draft", ci: domain.CIUnknown},
		{name: "merged", extra: `"isDraft":false,"state":"MERGED"`, token: "merged", ci: domain.CIUnknown},
		{name: "blocked-conflict", extra: `"isDraft":false,"state":"OPEN","mergeable":"CONFLICTING"`, token: "warning", ci: domain.CIUnknown},
		{name: "blocked-review", extra: `"isDraft":false,"state":"OPEN","reviewDecision":"CHANGES_REQUESTED"`, token: "warning", ci: domain.CIUnknown},
		{name: "queued", extra: `"isDraft":false,"state":"OPEN","mergeStateStatus":"QUEUED"`, token: "paper", ci: domain.CIUnknown},
		{name: "open-slim", extra: `"isDraft":false,"state":"OPEN"`, token: "paper", ci: domain.CIUnknown},
		{name: "draft-slim", extra: `"isDraft":true,"state":"OPEN"`, token: "draft", ci: domain.CIUnknown},
		{name: "approved-no-ci", extra: `"isDraft":false,"state":"OPEN","mergeable":"MERGEABLE","reviewDecision":"APPROVED","mergeStateStatus":"CLEAN"`, token: "ready", ci: domain.CIUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// A stack is 2+ PRs. The first layer carries the status fields.
			raw := `[{` + slimListPRJSON(1, "a", "main", tc.extra) + `},{` + slimListPRJSON(2, "b", "a", `"isDraft":false,"state":"OPEN"`) + `}]`
			run := ghOK(map[string][]byte{"pr list": []byte(raw)})
			stacks, err := fetchWith(run, "archetype-labs/app")
			if err != nil {
				t.Fatal(err)
			}
			if len(stacks) != 1 || len(stacks[0].PRs) != 2 {
				t.Fatalf("want one 2-PR stack, got %+v", stacks)
			}
			pr := stacks[0].PRs[0]
			state := domain.GetDisplayState(pr)
			if got := domain.StateColorToken(state); got != tc.token {
				t.Fatalf("token %q, want %q (state %s, pr %+v)", got, tc.token, state, pr)
			}
			if pr.CI.State != tc.ci {
				t.Fatalf("CI must stay %s without statusCheckRollup, got %s", tc.ci, pr.CI.State)
			}
		})
	}
}

func slimListPRJSON(number int, head, base, extra string) string {
	s := fmt.Sprintf(`"number":%d,"title":"layer","url":"https://github.com/owner/demo/pull/%d","headRefName":%q,"baseRefName":%q,"author":{"login":"gm"},"labels":[]`, number, number, head, base)
	if extra == "" {
		return s
	}
	return s + "," + extra
}

func TestFetchWithGroupsChainFromGhJSON(t *testing.T) {
	run := ghOK(map[string][]byte{
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
		t.Fatalf("want one chain of 2, got %+v", stacks)
	}
	if stacks[0].PRs[0].Number != 1 || stacks[0].PRs[1].Number != 2 {
		t.Fatalf("order %+v", numbers(stacks[0].PRs))
	}
	if stacks[0].PRs[0].Author != "gm" || stacks[0].PRs[0].AuthorColor != domain.LoginColor("gm") {
		t.Fatalf("author fallback: %+v", stacks[0].PRs[0])
	}
	if stacks[0].PRs[0].URL != "https://github.com/owner/demo/pull/1" {
		t.Fatalf("url %q", stacks[0].PRs[0].URL)
	}
	if stacks[0].PRs[0].Branch != "a" || stacks[0].PRs[1].Branch != "b" {
		t.Fatalf("headRefName mapping: %+v %+v", stacks[0].PRs[0], stacks[0].PRs[1])
	}
	if len(stacks[0].PRs[0].Labels) != 0 {
		t.Fatalf("no labels in this fixture: %+v", stacks[0].PRs[0].Labels)
	}
}

func TestFetchDropsOnePRStacks(t *testing.T) {
	run := ghOK(map[string][]byte{
		"pr list": []byte(`[
			{"number":1,"title":"solo","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","author":{"login":"gm"},"labels":[],"isDraft":false,"state":"OPEN"}
		]`),
	})
	stacks, err := fetchWith(run, "owner/demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 0 {
		t.Fatalf("1-PR is not a stack: %+v", stacks)
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

	run := ghOK(map[string][]byte{
		"pr list": []byte(`[
			{"number":1,"title":"bottom","url":"https://github.com/owner/demo/pull/1","headRefName":"a","baseRefName":"main","author":{"login":"gm","avatarUrl":"https://avatars.example/gm.png"},"labels":[{"name":"bug","color":"d73a4a"},{"name":"auth","color":"0e8a16"}],"isDraft":false,"state":"OPEN"},
			{"number":2,"title":"top","url":"https://github.com/owner/demo/pull/2","headRefName":"b","baseRefName":"a","author":{"login":"gm"},"labels":[],"isDraft":false,"state":"OPEN"}
		]`),
	})
	stacks, err := fetchWith(run, "owner/demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 1 || len(stacks[0].PRs) != 2 {
		t.Fatalf("want one 2-PR stack, got %+v", stacks)
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

	_, err := Fetch("archetype-labs/app")
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
	}, "archetype-labs/app")
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

func TestFetchDoesNotCallStacksOrRepoView(t *testing.T) {
	var calls []string
	run := func(args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		calls = append(calls, joined)
		if args[0] == "api" || strings.Contains(joined, "/pulls") || strings.Contains(joined, "stacks") {
			t.Fatalf("must not call gh api /pulls or /stacks: %v", args)
		}
		if args[0] == "repo" {
			t.Fatalf("must not call gh repo view: %v", args)
		}
		if args[0] != "pr" || args[1] != "list" {
			t.Fatalf("only gh pr list, got %v", args)
		}
		return []byte(`[]`), nil
	}
	if _, err := fetchWith(run, "archetype-labs/app"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 {
		t.Fatalf("one serial pr list, got %v", calls)
	}
}

func TestFetchRetries502OnceThenSucceeds(t *testing.T) {
	var slept []time.Duration
	oldSleep := sleep
	sleep = func(d time.Duration) { slept = append(slept, d) }
	t.Cleanup(func() { sleep = oldSleep })

	var listCalls atomic.Int32
	run := func(args ...string) ([]byte, error) {
		if args[0] != "pr" || args[1] != "list" {
			t.Fatalf("unexpected gh %v", args)
		}
		if listCalls.Add(1) == 1 {
			return nil, fmt.Errorf("gh pr list: HTTP 502: Bad Gateway (https://api.github.com/graphql)")
		}
		return []byte(`[]`), nil
	}
	if _, err := fetchWith(run, "archetype-labs/app"); err != nil {
		t.Fatalf("one 502 retry should succeed: %v", err)
	}
	if listCalls.Load() != 2 {
		t.Fatalf("second try should succeed, got %d calls", listCalls.Load())
	}
	if len(slept) != 1 {
		t.Fatalf("backoff before retry, got %v", slept)
	}
}

func TestFetchRetries502ThreeTimesThenSucceeds(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	var listCalls atomic.Int32
	run := func(args ...string) ([]byte, error) {
		if args[0] != "pr" || args[1] != "list" {
			t.Fatalf("unexpected gh %v", args)
		}
		if listCalls.Add(1) < 3 {
			return nil, fmt.Errorf("gh pr list: HTTP 502: Bad Gateway")
		}
		return []byte(`[]`), nil
	}
	if _, err := fetchWith(run, "archetype-labs/app"); err != nil {
		t.Fatalf("third try should succeed: %v", err)
	}
	if listCalls.Load() != 3 {
		t.Fatalf("three tries, got %d", listCalls.Load())
	}
}

func TestFetch502ThenErrorScreen(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls.Add(1)
		return nil, fmt.Errorf("HTTP 502: Bad Gateway (https://api.github.com/graphql)")
	}, "archetype-labs/app")
	if err == nil {
		t.Fatal("three 502s must fail")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Fatalf("keep the 502: %v", err)
	}
	if calls.Load() != retryLimit {
		t.Fatalf("three tries inside Fetch, got %d (limit %d)", calls.Load(), retryLimit)
	}
	if retryLimit != 3 {
		t.Fatalf("502/503 flake is 3 tries, not %d", retryLimit)
	}
}

func TestFetch503RetriesInsideFetch(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls.Add(1)
		return nil, fmt.Errorf("HTTP 503: Service Unavailable")
	}, "archetype-labs/app")
	if err == nil {
		t.Fatal("three 503s must fail")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Fatalf("keep the 503: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("503 is three tries inside Fetch, got %d", calls.Load())
	}
}

func TestFetch404IsNotRetried(t *testing.T) {
	sleep = func(time.Duration) { t.Fatal("404 must not backoff") }
	t.Cleanup(func() { sleep = time.Sleep })

	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls.Add(1)
		return nil, fmt.Errorf("gh pr list --repo archetype-labs/app --state open --limit 100: gh: Not Found (HTTP 404)")
	}, "archetype-labs/app")
	if err == nil {
		t.Fatal("404 must paper")
	}
	if errors.Is(err, ErrGHAuth) {
		t.Fatalf("404 is not auth: %v", err)
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("keep the 404: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("404 must not retry, calls=%d", calls.Load())
	}
}

func TestFetchAuthIsNotA502(t *testing.T) {
	sleep = func(time.Duration) { t.Fatal("auth must not backoff") }
	t.Cleanup(func() { sleep = time.Sleep })

	var calls atomic.Int32
	_, err := fetchWith(func(args ...string) ([]byte, error) {
		calls.Add(1)
		return nil, fmt.Errorf("gh pr list: HTTP 401: Bad credentials")
	}, "archetype-labs/app")
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

func TestGHCommandInheritsEnv(t *testing.T) {
	cmd := ghCommand(prListArgs("archetype-labs/app")...)
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
