package tui_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/cli"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/live"
	"github.com/gsimone/dango-tui/internal/summary"
	"github.com/gsimone/dango-tui/internal/tui"
)

func applyLiveFetch(m tui.Model) (tui.Model, tea.Cmd) {
	cmd := m.Init()
	if cmd == nil {
		return m, nil
	}
	next, extra := m.Update(cmd())
	return next.(tui.Model), extra
}

func applyLiveCmds(m tui.Model, cmd tea.Cmd) tui.Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = applyLiveCmds(m, c)
		}
		return m
	}
	next, extra := m.Update(msg)
	return applyLiveCmds(next.(tui.Model), extra)
}

func testdataJSON(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, "testdata", "test.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatal("testdata/test.json not found")
		}
		dir = next
	}
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func TestNoFlagDetectsOriginAndFetches(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "remote", "add", "origin", "https://github.com/owner/from-detect.git")

	parsed, err := cli.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	args, err := cli.Resolve(parsed, dir)
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "owner/from-detect" {
		t.Fatalf("detect %q", args.Repo)
	}

	fetches := 0
	m := tui.New(tui.Options{
		Repo:   args.Repo,
		Width:  80,
		Height: 24,
		Fetch: func(repo string) ([]domain.Stack, error) {
			fetches++
			if repo != "owner/from-detect" {
				t.Fatalf("fetched %q", repo)
			}
			return []domain.Stack{{
				ID: "s",
				PRs: []domain.PullRequest{
					{Number: 1, Title: "detected layer", Branch: "gm/detected"},
					{Number: 2, Title: "detected head", Branch: "gm/detected-head"},
				},
			}}, nil
		},
	})
	if fetches != 0 || !m.Live || m.File {
		t.Fatalf("constructor must not fetch, fetches=%d live=%v file=%v", fetches, m.Live, m.File)
	}
	if !strings.Contains(frameOf(m), "fetching owner/from-detect") {
		t.Fatalf("first frame before gh:\n%s", frameOf(m))
	}
	m, _ = applyLiveFetch(m)
	if fetches != 1 {
		t.Fatalf("Init fetches once, got %d", fetches)
	}
	if !strings.Contains(frameOf(m), "detected layer") || !strings.Contains(frameOf(m), "owner/from-detect") {
		t.Fatalf("live frame:\n%s", frameOf(m))
	}
	if strings.Contains(frameOf(m), "auth cleanup") || strings.Contains(frameOf(m), "300 stacks") {
		t.Fatalf("must not load examples:\n%s", frameOf(m))
	}
}

func TestNoFlagNoRemoteErrorsLoud(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")

	parsed, err := cli.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := cli.Resolve(parsed, dir)
	if err == nil {
		t.Fatal("no remote must die, not load examples")
	}
	if got.Repo != "" {
		t.Fatalf("failed resolve invented %q", got.Repo)
	}
	msg := err.Error()
	if !strings.Contains(msg, "--repo archetype-labs/app") || !strings.Contains(msg, "--repo testdata/test.json") {
		t.Fatalf("must name both --repo forms: %v", err)
	}
	m := tui.New(tui.Options{
		Repo:   got.Repo,
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("detect miss must not fetch or fall back")
			return nil, nil
		},
	})
	if m.Live || m.File || len(m.Stacks()) != 0 {
		t.Fatalf("empty repo is not examples, live=%v file=%v stacks=%d", m.Live, m.File, len(m.Stacks()))
	}
}

func TestLiveMissingGHShowsErrorNotFixtures(t *testing.T) {
	fetches := 0
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			fetches++
			return nil, live.ErrGHMissing
		},
	})
	if fetches != 0 {
		t.Fatalf("constructor must not fetch, got %d", fetches)
	}
	if !strings.Contains(frameOf(m), "fetching archetype-labs/app") {
		t.Fatalf("first frame before gh:\n%s", frameOf(m))
	}
	m, extra := applyLiveFetch(m)
	if extra != nil {
		t.Fatal("failed fetch stays on the splash")
	}
	if fetches != 1 {
		t.Fatalf("live --repo must fetch, got %d", fetches)
	}
	if !m.Live || m.Repo != "archetype-labs/app" {
		t.Fatalf("must stay live, got live=%v repo=%q", m.Live, m.Repo)
	}
	if n := len(m.Stacks()); n != 0 {
		t.Fatalf("must not load fixtures, got %d stacks", n)
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "gh CLI not found") || !strings.Contains(frame, "cli.github.com") {
		t.Fatalf("splash loading line becomes the error:\n%s", frame)
	}
	if strings.Contains(frame, "Could not fetch") || strings.Contains(frame, "No open stacks") {
		t.Fatalf("no empty-list error state:\n%s", frame)
	}
	if strings.Contains(frame, "org/reponame") {
		t.Fatalf("must not fall back to the fixture slug:\n%s", frame)
	}
	if strings.Contains(frame, "300 stacks") {
		t.Fatalf("must not load chaos fixtures:\n%s", frame)
	}
	if strings.Contains(frame, "YOINKS") {
		t.Fatalf("no joke words:\n%s", frame)
	}
}

func TestStoryFreightAndPairAreAuthored(t *testing.T) {
	freight := tui.New(tui.Options{StoryID: "freight", Width: 120, Height: 30})
	if freight.Live {
		t.Fatal("story is fixtures")
	}
	frame := frameOf(freight)
	if !strings.Contains(frame, "freight train") || !strings.Contains(frame, "Land the schema cutover") {
		t.Fatalf("freight demo:\n%s", frame)
	}
	if strings.Contains(frame, "Freight layer") || strings.Contains(frame, "300 stacks") {
		t.Fatalf("freight must stay authored:\n%s", frame)
	}

	pair := frameOf(tui.New(tui.Options{StoryID: "pair", Width: 80, Height: 24}))
	if !strings.Contains(pair, "Land the checkout helper") {
		t.Fatalf("pair demo:\n%s", pair)
	}
	if strings.Contains(pair, "Tiny left") {
		t.Fatalf("pair must stay authored:\n%s", pair)
	}
}

func TestStoryIgnoresLiveFetch(t *testing.T) {
	m := tui.New(tui.Options{
		StoryID: "mixed",
		Repo:    "gsimone/leva-2",
		Width:   80,
		Height:  24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("StoryID hook must ignore live fetch")
			return nil, nil
		},
	})
	if m.Live {
		t.Fatal("story is fixture path")
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "org/reponame  •  3 stacks / 8 layers") {
		t.Fatalf("fixture header:\n%s", frame)
	}
	if strings.Contains(frame, "gsimone/leva-2") {
		t.Fatalf("story must not paint the live slug:\n%s", frame)
	}
}

func TestLiveRepoHeaderAndTwoColumns(t *testing.T) {
	fetches := 0
	m := tui.New(tui.Options{
		Repo:   "gsimone/leva-2",
		Width:  120,
		Height: 30,
		Fetch: func(repo string) ([]domain.Stack, error) {
			fetches++
			if repo != "gsimone/leva-2" {
				t.Fatalf("repo %q", repo)
			}
			return []domain.Stack{{
				ID: "stack-1",
				PRs: []domain.PullRequest{
					{Number: 1, Title: "base", Branch: "a", URL: "https://github.com/gsimone/leva-2/pull/1"},
					{Number: 2, Title: "head", Branch: "b", URL: "https://github.com/gsimone/leva-2/pull/2", CI: domain.CISummary{State: domain.CIFailure, Failed: 1, Total: 1}},
				},
			}}, nil
		},
	})
	if fetches != 0 {
		t.Fatalf("constructor must not fetch, got %d", fetches)
	}
	first := frameOf(m)
	if !strings.Contains(first, "fetching gsimone/leva-2") || !strings.Contains(first, "●-●-●") {
		t.Fatalf("first frame is the splash:\n%s", first)
	}
	if strings.Contains(first, "●-●-● DANGO") {
		t.Fatalf("splash is not the list header:\n%s", first)
	}
	m, _ = applyLiveFetch(m)
	if fetches != 1 {
		t.Fatalf("initial fetch %d", fetches)
	}
	if !m.Live || m.Repo != "gsimone/leva-2" {
		t.Fatalf("live %+v repo %q", m.Live, m.Repo)
	}
	if m.Provider.Raw != "" {
		t.Fatalf("provider is optional, got %+v", m.Provider)
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "gsimone/leva-2  •  1 stacks / 2 layers") {
		t.Fatalf("live header:\n%s", frame)
	}
	if !strings.Contains(frame, "●-●") {
		t.Fatalf("list:\n%s", frame)
	}
	if !strings.Contains(strings.Join(listRows(frame), "\n"), "base") {
		t.Fatalf("list paints the gh name first:\n%s", frame)
	}
	if strings.Contains(frame, "base and head") {
		t.Fatalf("missing provider must not invent a stack title:\n%s", frame)
	}
	if strings.Contains(frame, "ci failed") {
		t.Fatalf("no status column:\n%s", frame)
	}
	for _, row := range listRows(frame) {
		for _, word := range []string{"pending", "ready", "blocked", "ci failed"} {
			if strings.Contains(row, word) {
				t.Fatalf("list row still has status word %q: %q", word, row)
			}
		}
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	m = next.(tui.Model)
	if !strings.Contains(frameOf(m), "⠋") {
		t.Fatalf("spinner:\n%s", frameOf(m))
	}
	if cmd == nil {
		t.Fatal("live r must call gh")
	}
	msg := cmd()
	next, _ = m.Update(msg)
	m = next.(tui.Model)
	if fetches != 2 {
		t.Fatalf("refresh fetch %d", fetches)
	}
	if !strings.Contains(frameOf(m), "last fetched just now") {
		t.Fatalf("relative fetch:\n%s", frameOf(m))
	}
}

func TestProviderWritesStackTitleOnly(t *testing.T) {
	fetch := func(string) ([]domain.Stack, error) {
		return []domain.Stack{{
			ID:  "s",
			PRs: []domain.PullRequest{{Number: 1, Title: "alpha layer"}, {Number: 2, Title: "beta layer"}},
		}}, nil
	}
	with := tui.New(tui.Options{
		Repo:     "owner/name",
		Provider: summary.ParseProvider("codex@luna.medium"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
	})
	if with.Provider.Name != "codex" || with.Provider.Model != "luna.medium" {
		t.Fatalf("store provider, got %+v", with.Provider)
	}
	with, sumCmd := applyLiveFetch(with)
	if sumCmd == nil {
		t.Fatal("provider must kick summary cmds after first paint")
	}
	first := listRows(frameOf(with))
	joined := strings.Join(first, "\n")
	if !strings.Contains(joined, "alpha layer") {
		t.Fatalf("list paints the gh name first:\n%s", joined)
	}
	if strings.Contains(joined, "alpha layer and beta layer") {
		t.Fatalf("first paint must not wait on a generated title:\n%s", joined)
	}
	if strings.Contains(frameOf(with), "⠋") {
		t.Fatalf("empty summary is not a spinner:\n%s", frameOf(with))
	}
	plain := tui.New(tui.Options{
		Repo:   "owner/name",
		Width:  80,
		Height: 24,
		Fetch:  fetch,
	})
	plain, extra := applyLiveFetch(plain)
	bare := strings.Join(listRows(frameOf(plain)), "\n")
	if !strings.Contains(bare, "alpha layer") {
		t.Fatalf("missing provider keeps the gh name:\n%s", bare)
	}
	if strings.Contains(bare, "alpha layer and beta layer") {
		t.Fatalf("first paint must not wait on a generated title:\n%s", bare)
	}
	plain = applyLiveCmds(plain, extra)
	bare = strings.Join(listRows(frameOf(plain)), "\n")
	if strings.Contains(bare, "alpha layer and beta layer") {
		t.Fatalf("missing provider must not invent a generated title:\n%s", bare)
	}
	if plain.Stacks()[0].Name != "alpha layer" {
		t.Fatalf("list name must stay the gh title: %+v", plain.Stacks()[0])
	}
	if plain.Stacks()[0].Description != "" {
		t.Fatalf("unset describe leaves the pane empty: %q", plain.Stacks()[0].Description)
	}
}

func TestLivePabloBallsOn120(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			pr := func(n int, title string, extra domain.PullRequest) domain.PullRequest {
				extra.Number = n
				extra.Title = title
				return extra
			}
			return []domain.Stack{
				{ID: "open", PRs: []domain.PullRequest{pr(1, "open base", domain.PullRequest{}), pr(2, "open head", domain.PullRequest{})}},
				{ID: "draft", PRs: []domain.PullRequest{pr(3, "draft base", domain.PullRequest{Draft: true, Mergeable: domain.MergeableFalse()}), pr(4, "draft head", domain.PullRequest{})}},
				{ID: "fail", PRs: []domain.PullRequest{pr(5, "fail base", domain.PullRequest{CI: domain.CISummary{State: domain.CIFailure, Failed: 1}}), pr(6, "fail head", domain.PullRequest{})}},
				{ID: "review", PRs: []domain.PullRequest{pr(7, "review base", domain.PullRequest{ReviewDecision: "CHANGES_REQUESTED"}), pr(8, "review head", domain.PullRequest{})}},
				{ID: "ok", PRs: []domain.PullRequest{pr(9, "approved base", domain.PullRequest{ReviewDecision: "APPROVED"}), pr(10, "approved head", domain.PullRequest{})}},
				{ID: "landed", PRs: []domain.PullRequest{pr(11, "merged base", domain.PullRequest{Merged: true}), pr(12, "merged head", domain.PullRequest{})}},
				{ID: "queue", PRs: []domain.PullRequest{pr(13, "queued base", domain.PullRequest{MergeQueueState: "QUEUED"}), pr(14, "queued head", domain.PullRequest{})}},
			}, nil
		},
	})
	m, _ = applyLiveFetch(m)
	frame := frameOf(m)
	t.Logf("120 Pablo balls:\n%s", frame)
	list := strings.Join(listRows(frame), "\n")
	if !strings.Contains(list, "▶") {
		t.Fatalf("selected stack is ▶:\n%s", frame)
	}
	if !strings.Contains(list, "●-○") {
		t.Fatalf("active is ● on a hollow chain:\n%s", frame)
	}
	if !strings.Contains(list, "◎-○") {
		t.Fatalf("needs-review stays ◎:\n%s", frame)
	}
	if !strings.Contains(list, "◌-○") {
		t.Fatalf("queued is ◌:\n%s", frame)
	}
	if !strings.Contains(list, "○-○") {
		t.Fatalf("idle layers are ○:\n%s", frame)
	}
	if strings.Contains(list, "◉") {
		t.Fatalf("fisheye is retired:\n%s", frame)
	}
	if strings.Count(list, "●") != 1 {
		t.Fatalf("filled never appears twice: %d in\n%s", strings.Count(list, "●"), frame)
	}
	if strings.Contains(list, "▸") {
		t.Fatalf("selected marker is ▶, not ▸:\n%s", frame)
	}
	rr, rg, rb, _ := domain.ParseRGB(domain.Color("surfaceRaised"))
	if strings.Contains(m.View(), fmt.Sprintf("48;2;%d;%d;%d", rr, rg, rb)) {
		t.Fatalf("selected row must not wash:\n%s", frame)
	}

	draft := applyKey(m, key("down"))
	draftList := strings.Join(listRows(frameOf(draft)), "\n")
	if !strings.Contains(draftList, "●-○") || strings.Count(draftList, "●") != 1 {
		t.Fatalf("draft you are on is one filled:\n%s", frameOf(draft))
	}
	for _, row := range listRows(frameOf(draft)) {
		if strings.Contains(row, "draft base") && strings.Contains(row, "◎") {
			t.Fatalf("unmergeable draft row is not review:\n%s", row)
		}
	}
	dr, dg, db, _ := domain.ParseRGB("#8b8e93")
	if !strings.Contains(draft.View(), fmt.Sprintf("38;2;%d;%d;%d", dr, dg, db)) {
		t.Fatal("draft you are on keeps meta gray #8b8e93")
	}
	if !strings.Contains(frameOf(draft), "status    draft") {
		t.Fatalf("inspector stays draft:\n%s", frameOf(draft))
	}

	fail := applyKey(applyKey(m, key("down")), key("down"))
	failList := strings.Join(listRows(frameOf(fail)), "\n")
	if !strings.Contains(failList, "●-○") || strings.Count(failList, "●") != 1 {
		t.Fatalf("fail you are on is one filled:\n%s", frameOf(fail))
	}
	fr, fg, fb, _ := domain.ParseRGB("#e24b4a")
	if !strings.Contains(fail.View(), fmt.Sprintf("38;2;%d;%d;%d", fr, fg, fb)) {
		t.Fatal("failing layer you are on keeps red ink")
	}

	review := applyKey(fail, key("down"))
	reviewList := strings.Join(listRows(frameOf(review)), "\n")
	if strings.Contains(reviewList, "◎-○") {
		t.Fatalf("active review is ●, not ◎:\n%s", frameOf(review))
	}
	if !strings.Contains(reviewList, "●-○") || strings.Count(reviewList, "●") != 1 {
		t.Fatalf("review you are on is one filled:\n%s", frameOf(review))
	}
	ar, ag, ab, _ := domain.ParseRGB("#e6b84d")
	if !strings.Contains(review.View(), fmt.Sprintf("38;2;%d;%d;%d", ar, ag, ab)) {
		t.Fatal("review layer you are on keeps amber ink")
	}
}

func TestLiveDraftChainOn120(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			return []domain.Stack{{
				ID: "s",
				PRs: []domain.PullRequest{
					{Number: 1, Title: "open layer"},
					{Number: 5209, Title: "draft layer", Draft: true, Mergeable: domain.MergeableFalse()},
					{Number: 3, Title: "review layer", ReviewDecision: "CHANGES_REQUESTED"},
				},
			}}, nil
		},
	})
	m, _ = applyLiveFetch(m)
	list := strings.Join(listRows(frameOf(m)), "\n")
	if !strings.Contains(list, "●-○-◎") {
		t.Fatalf("idle #5209 is ○ in the chain:\n%s", frameOf(m))
	}
	onDraft := applyKey(m, key("right"))
	onList := strings.Join(listRows(frameOf(onDraft)), "\n")
	if !strings.Contains(onList, "○-●-◎") {
		t.Fatalf("#5209 you are on is gray ● plus ○/◎:\n%s", frameOf(onDraft))
	}
	dr, dg, db, _ := domain.ParseRGB("#8b8e93")
	if !strings.Contains(onDraft.View(), fmt.Sprintf("38;2;%d;%d;%d", dr, dg, db)) {
		t.Fatal("draft you are on is meta gray")
	}
	if !strings.Contains(frameOf(onDraft), "status    draft") {
		t.Fatalf("unmergeable draft stays draft:\n%s", frameOf(onDraft))
	}
}

func TestLiveCIEnrichPaintsFailAfterList(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			return []domain.Stack{{
				ID: "s",
				PRs: []domain.PullRequest{
					{Number: 1, Title: "base layer"},
					{Number: 2, Title: "head layer"},
				},
			}}, nil
		},
		EnrichCI: func(repo string, stacks []domain.Stack) []domain.Stack {
			if repo != "archetype-labs/app" {
				t.Fatalf("repo %q", repo)
			}
			stacks[0].PRs[0].CI = domain.CISummary{State: domain.CIFailure, Failed: 1, Total: 2}
			return stacks
		},
	})
	m, extra := applyLiveFetch(m)
	first := frameOf(m)
	if domain.GetDisplayState(m.Stacks()[0].PRs[0]) == domain.StateCIFailure {
		t.Fatal("first paint must not wait on checks")
	}
	m = applyLiveCmds(m, extra)
	if domain.GetDisplayState(m.Stacks()[0].PRs[0]) != domain.StateCIFailure {
		t.Fatalf("enrich must paint fail: %+v", m.Stacks()[0].PRs[0])
	}
	if !strings.Contains(frameOf(m), "CI failing") {
		t.Fatalf("inspector after enrich:\n%s", frameOf(m))
	}
	if strings.Contains(first, "statusCheckRollup") {
		t.Fatal("first list must stay off rollup")
	}
}

func TestLiveDescribeScriptWithoutProvider(t *testing.T) {
	title := "LEV-182: Bound hosts to the session so undo does not wedge"
	scripted := "script pinned hosts to the worker so undo cannot widen scope"
	var jobs []summary.Job
	m := tui.New(tui.Options{
		Repo:     "archetype-labs/app",
		Describe: "bin/describe-stack",
		Width:    120,
		Height:   30,
		Fetch: func(string) ([]domain.Stack, error) {
			return []domain.Stack{{
				ID: "s",
				PRs: []domain.PullRequest{
					{Number: 182, Title: title},
					{Number: 183, Title: "Pin each host to the worker"},
				},
			}}, nil
		},
		Summarize: func(job summary.Job) summary.Result {
			jobs = append(jobs, job)
			if job.Provider.Raw != "" {
				t.Fatalf("no provider: %+v", job.Provider)
			}
			if job.Describe != "bin/describe-stack" {
				t.Fatalf("describe command: %q", job.Describe)
			}
			return summary.Result{ID: job.ID, Description: scripted}
		},
	})
	m, extra := applyLiveFetch(m)
	if extra == nil {
		t.Fatal("describe must start after first paint, not block fetch")
	}
	first := frameOf(m)
	list := strings.Join(listRows(first), "\n")
	if !strings.Contains(list, "LEV-182") {
		t.Fatalf("first paint keeps the short list title:\n%s", first)
	}
	if strings.Contains(first, scripted) {
		t.Fatalf("describe must not block first paint:\n%s", first)
	}
	local := summary.Describe(m.Stacks()[0])
	m = applyLiveCmds(m, extra)
	if len(jobs) != 1 {
		t.Fatalf("one process for the selected stack, got %d", len(jobs))
	}
	if m.Stacks()[0].Name != "LEV-182" {
		t.Fatalf("list title unchanged, got %q", m.Stacks()[0].Name)
	}
	if m.Stacks()[0].Description != scripted {
		t.Fatalf("script description: %q", m.Stacks()[0].Description)
	}
	if m.Stacks()[0].Description == local && local != "" {
		t.Fatal("Describe() is not the product sentence")
	}
	frame := frameOf(m)
	if strings.Contains(strings.Join(listRows(frame), "\n"), scripted) {
		t.Fatalf("description belongs in the pane:\n%s", frame)
	}
	if !strings.Contains(frame, "script pinned hosts") {
		t.Fatalf("pane must show script:\n%s", frame)
	}
}

func TestLiveDescribeSelectedStackFirst(t *testing.T) {
	var ids []string
	m := tui.New(tui.Options{
		Repo:     "archetype-labs/app",
		Describe: "bin/describe-stack",
		Width:    120,
		Height:   30,
		Fetch: func(string) ([]domain.Stack, error) {
			return []domain.Stack{
				{ID: "a", PRs: []domain.PullRequest{{Number: 1, Title: "alpha base"}, {Number: 2, Title: "alpha head"}}},
				{ID: "b", PRs: []domain.PullRequest{{Number: 3, Title: "beta base"}, {Number: 4, Title: "beta head"}}},
				{ID: "c", PRs: []domain.PullRequest{{Number: 5, Title: "gamma base"}, {Number: 6, Title: "gamma head"}}},
			}, nil
		},
		Summarize: func(job summary.Job) summary.Result {
			ids = append(ids, job.ID)
			return summary.Result{ID: job.ID, Description: "desc-" + job.ID}
		},
	})
	m, extra := applyLiveFetch(m)
	if extra == nil {
		t.Fatal("selected describe starts after first paint")
	}
	m = applyLiveCmds(m, extra)
	if strings.Join(ids, ",") != "a" {
		t.Fatalf("selected first, one process: %v", ids)
	}
	if m.Stacks()[0].Description != "desc-a" {
		t.Fatalf("selected landed: %+v", m.Stacks()[0])
	}
	if m.Stacks()[1].Description != "" || m.Stacks()[2].Description != "" {
		t.Fatalf("must not spawn per row: %+v", m.Stacks())
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = applyLiveCmds(next.(tui.Model), cmd)
	if strings.Join(ids, ",") != "a,b" {
		t.Fatalf("next selected after first finishes: %v", ids)
	}
	if m.Stacks()[1].Description != "desc-b" {
		t.Fatalf("second selected: %+v", m.Stacks()[1])
	}
	if m.Stacks()[2].Description != "" {
		t.Fatalf("still not per-row: %+v", m.Stacks()[2])
	}
}

func TestLiveEmptyPaneWithoutDescribe(t *testing.T) {
	title := "LEV-182: Bound hosts to the session so undo does not wedge"
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			return []domain.Stack{{
				ID: "s",
				PRs: []domain.PullRequest{
					{Number: 182, Title: title, Body: "<!-- CURSOR_AGENT_PR_BODY_BEGIN -->\nPin each bound host to the worker.\n"},
					{Number: 183, Title: "Pin each host to the worker"},
				},
			}}, nil
		},
	})
	if !strings.Contains(frameOf(m), "fetching archetype-labs/app") {
		t.Fatalf("splash first:\n%s", frameOf(m))
	}
	m, sumCmd := applyLiveFetch(m)
	if m.Provider.Raw != "" || m.Describe != "" {
		t.Fatal("no dango.json describe / --provider")
	}
	first := frameOf(m)
	list := strings.Join(listRows(first), "\n")
	if !strings.Contains(list, "LEV-182") {
		t.Fatalf("list still paints:\n%s", first)
	}
	if strings.Contains(list, "and pin each") {
		t.Fatalf("first paint must not invent a list title:\n%s", list)
	}
	if strings.Contains(first, "CURSOR_AGENT") || strings.Contains(first, "Covers ") || strings.Contains(first, "Pin each bound host to the worker") {
		t.Fatalf("first paint leaked body:\n%s", first)
	}
	m = applyLiveCmds(m, sumCmd)
	if m.Stacks()[0].Name != "LEV-182" {
		t.Fatalf("list title must stay short, got %q", m.Stacks()[0].Name)
	}
	if m.Stacks()[0].Description != "" {
		t.Fatalf("unset describe leaves the pane empty, not Describe(): %q", m.Stacks()[0].Description)
	}
	frame := frameOf(m)
	if strings.Contains(frame, "CURSOR_AGENT") || strings.Contains(frame, "Covers ") || strings.Contains(frame, "Pin each bound host to the worker") {
		t.Fatalf("body / Covers leaked:\n%s", frame)
	}
}

func TestFileRepoKeepsAuthoredNamesWithProvider(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:     testdataJSON(t),
		Provider: summary.ParseProvider("codex@luna.medium"),
		Width:    120,
		Height:   30,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("JSON --repo must not call gh")
			return nil, nil
		},
	})
	if m.Init() != nil {
		t.Fatal("authored dump must not start live summaries")
	}
	frame := frameOf(m)
	if !strings.Contains(strings.Join(listRows(frame), "\n"), "auth cleanup") {
		t.Fatalf("file names stay authored:\n%s", frame)
	}
	if strings.Contains(frame, "split auth scope from session checks, keep") {
		t.Fatalf("provider must not rewrite a dump title:\n%s", frame)
	}
}

func listRows(frame string) []string {
	var out []string
	for _, line := range strings.Split(frame, "\n") {
		part := line
		if idx := strings.Index(line, "│"); idx >= 0 {
			part = line[:idx]
		}
		if strings.Contains(part, "▶") || strings.Contains(part, "▸") || strings.Contains(part, "·") {
			out = append(out, part)
		}
	}
	return out
}

func TestStackedCardDoesNotEatTheList(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   testdataJSON(t),
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("JSON --repo must not call gh")
			return nil, nil
		},
	})
	frame := frameOf(m)
	if !strings.Contains(frame, "┌") || !strings.Contains(frame, "└") {
		t.Fatalf("stacked card:\n%s", frame)
	}
	rows := listRows(frame)
	joined := strings.Join(rows, "\n")
	for _, name := range []string{"auth cleanup", "composer tokens", "sync rewrite", "schema cutover"} {
		if !strings.Contains(joined, name) {
			t.Fatalf("short card must leave the list visible, missing %q:\n%s", name, frame)
		}
	}
	top, bot := -1, -1
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		if strings.Contains(line, "┌") {
			top = i
		}
		if strings.Contains(line, "└") {
			bot = i
		}
	}
	if top < 0 || bot < 0 || bot-top > 16 {
		t.Fatalf("stacked card is leftover-tall (%d..%d):\n%s", top, bot, frame)
	}
	composerAt := -1
	for i, line := range lines {
		if strings.Contains(line, "composer tokens") {
			composerAt = i
			break
		}
	}
	if composerAt < 0 || composerAt < bot {
		t.Fatalf("next stacks should sit under a short card, composer=%d bot=%d:\n%s", composerAt, bot, frame)
	}
}

func TestRepoJSONFilePaintsAuthoredStacks(t *testing.T) {
	path := testdataJSON(t)
	parsed, err := cli.Parse([]string{"--repo", path})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	gitIn(t, dir, "remote", "add", "origin", "https://github.com/owner/from-detect.git")
	args, err := cli.Resolve(parsed, dir)
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != path {
		t.Fatalf("--repo file must win over detect, got %q", args.Repo)
	}

	fetches := 0
	m := tui.New(tui.Options{
		Repo:   args.Repo,
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			fetches++
			t.Fatal("JSON --repo must not call gh")
			return nil, nil
		},
	})
	if fetches != 0 {
		t.Fatalf("file path must not fetch, got %d", fetches)
	}
	if m.Live || !m.File {
		t.Fatalf("file mode live=%v file=%v", m.Live, m.File)
	}
	if m.Repo != "example/stacks" {
		t.Fatalf("header slug from dump, got %q", m.Repo)
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "example/stacks  •  4 stacks / 18 layers") {
		t.Fatalf("authored header:\n%s", frame)
	}
	if !strings.Contains(frame, "●-●-● DANGO") || strings.Contains(frame, "🍡") {
		t.Fatalf("mark:\n%s", frame)
	}
	rows := strings.Join(listRows(frame), "\n")
	for _, name := range []string{"auth cleanup", "composer tokens", "sync rewrite", "schema cutover"} {
		if !strings.Contains(rows, name) {
			t.Fatalf("missing %q:\n%s", name, rows)
		}
	}
	if strings.Contains(rows, "Freight layer 1") || strings.Contains(frame, "300 stacks") {
		t.Fatalf("must not load chaos/random fixtures:\n%s", frame)
	}
	if !strings.Contains(frame, "labels    bug auth") {
		t.Fatalf("testdata labels stay authored:\n%s", frame)
	}
	if !strings.Contains(frame, "author    ● gianni") {
		t.Fatalf("testdata author row:\n%s", frame)
	}
	first := m.Stacks()[0].PRs[0]
	if first.Labels[0].Color != "#d73a4a" || first.Labels[1].Color != "#0e8a16" {
		t.Fatalf("keep testdata hexes: %+v", first.Labels)
	}
	if domain.IsLowChromaHex(first.AuthorColor) {
		t.Fatalf("--repo testdata author ● is grey: %s", first.AuthorColor)
	}
	if len(m.Stacks()[0].PRs[1].Labels) != 0 {
		t.Fatalf("do not invent labels on unlabeled testdata PRs: %+v", m.Stacks()[0].PRs[1].Labels)
	}
	if strings.Contains(frame, "[ p ]") {
		t.Fatalf("no picker:\n%s", frame)
	}
}

func TestRepoJSONMissingFileIsErrorNotFixtures(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   filepath.Join(t.TempDir(), "missing.json"),
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("missing json must not fetch")
			return nil, nil
		},
	})
	if m.Live || !m.File {
		t.Fatal("missing json stays file mode")
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "read ") || !strings.Contains(frame, "missing.json") {
		t.Fatalf("loud file error:\n%s", frame)
	}
	if strings.Contains(frame, "org/reponame") || strings.Contains(frame, "300 stacks") {
		t.Fatalf("must not fall back to fixtures:\n%s", frame)
	}
}

func TestLiveArchetypeAppNoOneBallRows(t *testing.T) {
	if os.Getenv("DANGO_LIVE_PROOF") == "" {
		t.Skip("manual: DANGO_LIVE_PROOF=1 go test ./internal/tui -run TestLiveArchetypeAppNoOneBallRows")
	}
	stacks, err := live.Fetch("archetype-labs/app")
	if err != nil {
		t.Fatalf("live Fetch archetype-labs/app: %v", err)
	}
	ones := 0
	layers := 0
	for i, stack := range stacks {
		layers += len(stack.PRs)
		if len(stack.PRs) < 2 {
			ones++
			t.Errorf("one-ball row %d %q n=%d", i, stack.Name, len(stack.PRs))
		}
	}
	if ones > 0 {
		t.Fatalf("live list still has %d one-ball rows", ones)
	}
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			return stacks, nil
		},
	})
	m, _ = applyLiveFetch(m)
	frame := frameOf(m)
	t.Logf("live archetype-labs/app: %d stacks / %d layers\n%s", len(m.Stacks()), layers, frame)
}

func TestLiveDropsOnePRStacksKeepsShortListTitle(t *testing.T) {
	title := "LEV-182: Bound hosts to the session so undo does not wedge"
	m := tui.New(tui.Options{
		Repo:   "archetype-labs/app",
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			return []domain.Stack{
				{ID: "solo", Name: "LEV-182", PRs: []domain.PullRequest{{Number: 1, Title: "solo open PR"}}},
				{ID: "s", Name: "LEV-182", PRs: []domain.PullRequest{
					{Number: 182, Title: title, Branch: "gm/bound"},
					{Number: 183, Title: "head layer", Branch: "gm/bound-head"},
				}},
			}, nil
		},
	})
	m, _ = applyLiveFetch(m)
	if len(m.Stacks()) != 1 || len(m.Stacks()[0].PRs) != 2 {
		t.Fatalf("1-PR row must not appear: %+v", m.Stacks())
	}
	frame := frameOf(m)
	if strings.Contains(frame, "solo open PR") {
		t.Fatalf("one-ball stack leaked:\n%s", frame)
	}
	list := strings.Join(listRows(frame), "\n")
	if strings.Contains(list, "Bound hosts") || strings.Contains(list, "does not wedge") {
		t.Fatalf("list must stay the short title:\n%s", list)
	}
	if !strings.Contains(list, "LEV-182") {
		t.Fatalf("list keeps the ticket:\n%s", frame)
	}
	if !strings.Contains(frame, "Bound hosts") || !strings.Contains(frame, "session") {
		t.Fatalf("pane must show the GitHub title:\n%s", frame)
	}
	if !strings.Contains(frame, "archetype-labs/app  •  1 stacks / 2 layers") {
		t.Fatalf("counts are real stacks only:\n%s", frame)
	}
}

func TestRepoOwnerNameStillFetches(t *testing.T) {
	fetches := 0
	m := tui.New(tui.Options{
		Repo:   "gsimone/leva-2",
		Width:  80,
		Height: 24,
		Fetch: func(repo string) ([]domain.Stack, error) {
			fetches++
			if repo != "gsimone/leva-2" {
				t.Fatalf("repo %q", repo)
			}
			return []domain.Stack{{
				ID: "s",
				PRs: []domain.PullRequest{
					{Number: 1, Title: "live layer"},
					{Number: 2, Title: "live head"},
				},
			}}, nil
		},
	})
	if fetches != 0 || !m.Live || m.File {
		t.Fatalf("constructor must not fetch, fetches=%d live=%v file=%v", fetches, m.Live, m.File)
	}
	m, _ = applyLiveFetch(m)
	if fetches != 1 {
		t.Fatalf("owner/name is live gh, fetches=%d", fetches)
	}
	if !strings.Contains(frameOf(m), "live layer") {
		t.Fatalf("live list:\n%s", frameOf(m))
	}
}

func TestDotCopiesTestdataBranchToast(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   testdataJSON(t),
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("JSON --repo must not call gh")
			return nil, nil
		},
	})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(tui.Model)
	frame := frameOf(m)
	if !strings.Contains(frame, "copied gm/auth-scope") {
		t.Fatalf("dot toast:\n%s", frame)
	}
	if strings.Contains(frame, "Checked out") || strings.Contains(frame, "[ p ]") {
		t.Fatalf("toast only, no checkout/picker:\n%s", frame)
	}
	if cmd == nil {
		t.Fatal("toast should clear")
	}
}

func TestFixtureRefreshStaysSimulated(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("r"))
	if !strings.Contains(frameOf(m), "⠋") {
		t.Fatalf("fixture spinner:\n%s", frameOf(m))
	}
}
