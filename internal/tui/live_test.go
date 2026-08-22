package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/cli"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/tui"
)

func TestNeedRepoAsksForFlag(t *testing.T) {
	m := tui.New(tui.Options{
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			t.Fatal("empty path must not fetch")
			return nil, nil
		},
	})
	if !m.NeedRepo || m.Live {
		t.Fatalf("want empty/help, live=%v need=%v", m.Live, m.NeedRepo)
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "pass --repo owner/name") {
		t.Fatalf("header should tell the user to pass --repo:\n%s", frame)
	}
	if !strings.Contains(frame, "Pass --repo owner/name to load live stacks.") {
		t.Fatalf("empty help:\n%s", frame)
	}
	if strings.Contains(frame, "org/reponame") {
		t.Fatalf("do not invent a default repo:\n%s", frame)
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
	if m.Live || m.NeedRepo {
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
		Repo:     "gsimone/leva-2",
		Provider: cli.ParseProvider("codex@luna.medium"),
		Width:    120,
		Height:   30,
		Fetch: func(repo string) ([]domain.Stack, error) {
			fetches++
			if repo != "gsimone/leva-2" {
				t.Fatalf("repo %q", repo)
			}
			return []domain.Stack{{
				ID:   "stack-1",
				Name: "live chain",
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
	if m.Provider.Name != "codex" || m.Provider.Model != "luna.medium" {
		t.Fatalf("provider %+v", m.Provider)
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "gsimone/leva-2  •  1 stacks / 2 layers") {
		t.Fatalf("live header:\n%s", frame)
	}
	if !strings.Contains(frame, "live chain") || !strings.Contains(frame, "○") {
		t.Fatalf("list:\n%s", frame)
	}
	if strings.Contains(frame, "ci failed") {
		t.Fatalf("no status column:\n%s", frame)
	}
	var row string
	for _, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, "live chain") {
			row = line
			break
		}
	}
	listPart := row
	if idx := strings.Index(row, "│"); idx >= 0 {
		listPart = row[:idx]
	}
	for _, word := range []string{"pending", "ready", "blocked", "ci failed"} {
		if strings.Contains(listPart, word) {
			t.Fatalf("list row still has status word %q: %q", word, listPart)
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

func TestFixtureRefreshStaysSimulated(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("r"))
	if !strings.Contains(frameOf(m), "⠋") {
		t.Fatalf("fixture spinner:\n%s", frameOf(m))
	}
}
