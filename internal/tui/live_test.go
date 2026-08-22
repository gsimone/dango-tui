package tui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/live"
	"github.com/gsimone/dango-tui/internal/summary"
	"github.com/gsimone/dango-tui/internal/tui"
)

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

func TestNoRepoUsesFixture(t *testing.T) {
	m := tui.New(tui.Options{
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("fixture path must not fetch")
			return nil, nil
		},
	})
	if m.Live {
		t.Fatal("no --repo is fixtures")
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "●-●-● DANGO") || strings.Contains(frame, "🍡") {
		t.Fatalf("same chrome as live:\n%s", frame)
	}
	if !strings.Contains(frame, "org/reponame") {
		t.Fatalf("fixture header slug:\n%s", frame)
	}
	if !strings.Contains(frame, "last fetched 2 mins ago") {
		t.Fatalf("same fetch chrome as live:\n%s", frame)
	}
	if strings.Contains(frame, "DEMO") || strings.Contains(frame, "demo") {
		t.Fatalf("no demo theme:\n%s", frame)
	}
	for _, name := range []string{"auth cleanup", "pair", "freight train"} {
		if !strings.Contains(frame, name) {
			t.Fatalf("default examples include %q:\n%s", name, frame)
		}
	}
	if !strings.Contains(frame, "5 stacks / 30 layers") {
		t.Fatalf("mixed+pair+freight counts:\n%s", frame)
	}
	if strings.Contains(frame, "300 stacks") {
		t.Fatalf("default examples must not be chaos:\n%s", frame)
	}
	if strings.Contains(frame, "pass --repo") {
		t.Fatalf("no empty/help when examples load:\n%s", frame)
	}
	if strings.Contains(frame, "[ p ]") {
		t.Fatalf("no picker:\n%s", frame)
	}
}

func TestLiveMissingGHShowsErrorNotFixtures(t *testing.T) {
	fetches := 0
	m := tui.New(tui.Options{
		Repo:   "owner/name",
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			fetches++
			return nil, live.ErrGHMissing
		},
	})
	if fetches != 1 {
		t.Fatalf("live --repo must fetch, got %d", fetches)
	}
	if !m.Live || m.Repo != "owner/name" {
		t.Fatalf("must stay live, got live=%v repo=%q", m.Live, m.Repo)
	}
	if n := len(m.Stacks()); n != 0 {
		t.Fatalf("must not load fixtures, got %d stacks", n)
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "gh CLI not found") || !strings.Contains(frame, "cli.github.com") {
		t.Fatalf("error pane must show the loud gh sentence:\n%s", frame)
	}
	if strings.Contains(frame, "org/reponame") {
		t.Fatalf("must not fall back to the fixture slug:\n%s", frame)
	}
	if strings.Contains(frame, "300 stacks") {
		t.Fatalf("must not load chaos fixtures:\n%s", frame)
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
	if !strings.Contains(frame, "○") {
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
	if with.Init() == nil {
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
	if plain.Init() != nil {
		t.Fatal("missing provider must not start summaries")
	}
	bare := strings.Join(listRows(frameOf(plain)), "\n")
	if !strings.Contains(bare, "alpha layer") {
		t.Fatalf("missing provider keeps the gh name:\n%s", bare)
	}
	if strings.Contains(bare, "alpha layer and beta layer") {
		t.Fatalf("missing provider must not invent a generated title:\n%s", bare)
	}
}

func listRows(frame string) []string {
	var out []string
	for _, line := range strings.Split(frame, "\n") {
		part := line
		if idx := strings.Index(line, "│"); idx >= 0 {
			part = line[:idx]
		}
		if strings.Contains(part, "▸") || strings.Contains(part, "·") {
			out = append(out, part)
		}
	}
	return out
}

func TestRepoJSONFilePaintsAuthoredStacks(t *testing.T) {
	fetches := 0
	m := tui.New(tui.Options{
		Repo:   testdataJSON(t),
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
				ID:  "s",
				PRs: []domain.PullRequest{{Number: 1, Title: "live layer"}},
			}}, nil
		},
	})
	if fetches != 1 || !m.Live || m.File {
		t.Fatalf("owner/name is live gh, fetches=%d live=%v file=%v", fetches, m.Live, m.File)
	}
	if !strings.Contains(frameOf(m), "live layer") {
		t.Fatalf("live list:\n%s", frameOf(m))
	}
}

func TestCommaCopiesTestdataBranchToast(t *testing.T) {
	m := tui.New(tui.Options{
		Repo:   testdataJSON(t),
		Width:  120,
		Height: 30,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("JSON --repo must not call gh")
			return nil, nil
		},
	})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	m = next.(tui.Model)
	frame := frameOf(m)
	if !strings.Contains(frame, "copied gm/auth-scope") {
		t.Fatalf("comma toast:\n%s", frame)
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
