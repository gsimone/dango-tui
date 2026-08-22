package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/summary"
	"github.com/gsimone/dango-tui/internal/tui"
)

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
	if !strings.Contains(frame, "org/reponame") {
		t.Fatalf("fixture header slug:\n%s", frame)
	}
	if strings.Contains(frame, "pass --repo") {
		t.Fatalf("no empty/help when fixtures load:\n%s", frame)
	}
}

func TestStoryIgnoresLiveFetch(t *testing.T) {
	m := tui.New(tui.Options{
		StoryID: "mixed",
		Repo:    "gsimone/leva-2",
		Width:   80,
		Height:  24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("-story must ignore live fetch")
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
	if strings.Contains(strings.Join(first, "\n"), "alpha layer") {
		t.Fatalf("first paint must not wait on the summarizer:\n%s", first)
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
	bare := listRows(frameOf(plain))
	joined := strings.Join(bare, "\n")
	if strings.Contains(joined, "alpha layer") || strings.Contains(joined, "beta layer") {
		t.Fatalf("missing provider must not invent a stack title:\n%s", joined)
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

func TestFixtureRefreshStaysSimulated(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("r"))
	if !strings.Contains(frameOf(m), "⠋") {
		t.Fatalf("fixture spinner:\n%s", frameOf(m))
	}
}
