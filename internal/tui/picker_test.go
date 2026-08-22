package tui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/summary"
	"github.com/gsimone/dango-tui/internal/tui"
)

func TestPickerOpenSelectEsc(t *testing.T) {
	m := makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed")
	if m.Picking {
		t.Fatal("picker must not open on first paint")
	}
	idle := frameOf(m)
	if strings.Contains(idle, "codex@luna.medium") {
		t.Fatalf("catalog is not on the stack list:\n%s", idle)
	}
	if !strings.Contains(idle, "[ p ]") {
		t.Fatalf("footer should advertise p:\n%s", idle)
	}
	if strings.Contains(idle, "[ enter ]") {
		t.Fatalf("enter stays off the stack footer:\n%s", idle)
	}

	m = applyKey(m, key("p"))
	if !m.Picking {
		t.Fatal("p opens the picker")
	}
	open := frameOf(m)
	for _, needle := range []string{"none", "local", "codex@luna.medium", "codex@luna.high"} {
		if !strings.Contains(open, needle) {
			t.Fatalf("picker missing %q:\n%s", needle, open)
		}
	}
	if strings.Contains(open, "auth cleanup") {
		t.Fatalf("stack list must leave:\n%s", open)
	}
	if !strings.Contains(open, "[ enter ]") || !strings.Contains(open, "[ esc ]") {
		t.Fatalf("picker footer:\n%s", open)
	}
	if !strings.Contains(open, "▸ none") {
		t.Fatalf("empty provider preselects none:\n%s", open)
	}

	m = applyKey(m, key("esc"))
	if m.Picking {
		t.Fatal("esc closes the picker")
	}
	if m.Provider.Raw != "" || m.Provider.Name != "" {
		t.Fatalf("esc must not change provider, got %+v", m.Provider)
	}
	closed := frameOf(m)
	if !strings.Contains(closed, "auth cleanup") {
		t.Fatalf("esc returns to the stack list:\n%s", closed)
	}
	if strings.Contains(closed, "codex@luna.medium") || strings.Contains(closed, "▸ none") {
		t.Fatalf("catalog must leave:\n%s", closed)
	}

	m = applyKey(m, key("p"))
	m = applyKey(m, key("down"))
	m = applyKey(m, key("down"))
	selected := frameOf(m)
	if !strings.Contains(selected, "▸ codex@luna.medium") {
		t.Fatalf("down to the example provider:\n%s", selected)
	}
	m = applyKey(m, key("enter"))
	if m.Picking {
		t.Fatal("enter closes the picker")
	}
	if m.Provider.Raw != "codex@luna.medium" || m.Provider.Name != "codex" || m.Provider.Model != "luna.medium" {
		t.Fatalf("enter selects the provider, got %+v", m.Provider)
	}
	back := frameOf(m)
	if !strings.Contains(back, "auth cleanup") {
		t.Fatalf("enter returns to the stack list:\n%s", back)
	}
	if strings.Contains(back, "▸ none") {
		t.Fatalf("picker must leave after select:\n%s", back)
	}
}

func TestProviderFlagSkipsPickerAndPreselects(t *testing.T) {
	fetches := 0
	fetch := func(string) ([]domain.Stack, error) {
		fetches++
		return []domain.Stack{{
			ID:  "s",
			PRs: []domain.PullRequest{{Number: 1, Title: "alpha layer"}, {Number: 2, Title: "beta layer"}},
		}}, nil
	}
	m := tui.New(tui.Options{
		Repo:     "owner/name",
		Provider: summary.ParseProvider("codex@luna.medium"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
	})
	if fetches != 1 {
		t.Fatalf("flag must not skip fetch, got %d", fetches)
	}
	if m.Picking {
		t.Fatal("--provider skips the picker screen")
	}
	if m.Init() == nil {
		t.Fatal("flag still kicks summaries after first paint")
	}
	first := frameOf(m)
	if !strings.Contains(strings.Join(listRows(first), "\n"), "alpha layer") {
		t.Fatalf("first paint is the stack list:\n%s", first)
	}
	if strings.Contains(first, "▸ none") || strings.Contains(first, "codex@luna.high") {
		t.Fatalf("picker must not replace first paint:\n%s", first)
	}

	next, cmd := m.Update(key("p"))
	m = next.(tui.Model)
	if cmd != nil {
		t.Fatal("opening the picker is not a fetch")
	}
	open := frameOf(m)
	if !m.Picking || !strings.Contains(open, "▸ codex@luna.medium") {
		t.Fatalf("flag preselects the current provider:\n%s", open)
	}
	if strings.Contains(open, "alpha layer") {
		t.Fatalf("picker is a full-screen list:\n%s", open)
	}

	next, cmd = m.Update(key("enter"))
	m = next.(tui.Model)
	if m.Picking {
		t.Fatal("enter closes")
	}
	if m.Provider.Raw != "codex@luna.medium" {
		t.Fatalf("reselecting the flag keeps it, got %+v", m.Provider)
	}
	if cmd != nil {
		t.Fatal("same provider must not restart summaries")
	}

	custom := tui.New(tui.Options{
		Repo:     "owner/name",
		Provider: summary.ParseProvider("flag@model"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
	})
	if custom.Picking {
		t.Fatal("unknown --provider still skips the screen")
	}
	custom = applyKey(custom, key("p"))
	if !strings.Contains(frameOf(custom), "▸ flag@model") {
		t.Fatalf("flag value not in the catalog is still preselected:\n%s", frameOf(custom))
	}
}

func TestPickerSelectKicksSummaries(t *testing.T) {
	fetch := func(string) ([]domain.Stack, error) {
		return []domain.Stack{{
			ID:  "s",
			PRs: []domain.PullRequest{{Number: 1, Title: "alpha layer"}},
		}}, nil
	}
	m := tui.New(tui.Options{
		Repo:   "owner/name",
		Width:  80,
		Height: 24,
		Fetch:  fetch,
	})
	if m.Init() != nil {
		t.Fatal("missing provider must not start summaries")
	}
	next, _ := m.Update(key("p"))
	m = next.(tui.Model)
	next, _ = m.Update(key("down"))
	m = next.(tui.Model)
	next, _ = m.Update(key("down"))
	m = next.(tui.Model)
	next, cmd := m.Update(key("enter"))
	m = next.(tui.Model)
	if m.Provider.Raw != "codex@luna.medium" {
		t.Fatalf("selected %+v", m.Provider)
	}
	if cmd == nil {
		t.Fatal("selecting a provider must kick summaries")
	}

	next, _ = m.Update(key("p"))
	m = next.(tui.Model)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	m = next.(tui.Model)
	if !strings.Contains(frameOf(m), "▸ none") {
		t.Fatalf("home should land on none:\n%s", frameOf(m))
	}
	next, cmd = m.Update(key("enter"))
	m = next.(tui.Model)
	if !m.Provider.Empty() {
		t.Fatalf("none clears the provider, got %+v", m.Provider)
	}
	if cmd != nil {
		t.Fatal("none must not start summaries")
	}
}
