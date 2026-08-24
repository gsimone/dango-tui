package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/live"
)

func TestDangoBlockIsThreeShadeRows(t *testing.T) {
	if len(dangoBlock) != 3 {
		t.Fatalf("3-row letters, got %d", len(dangoBlock))
	}
	width := displayWidth(dangoBlock[0])
	for i, row := range dangoBlock {
		if displayWidth(row) != width {
			t.Fatalf("row %d width %d, want %d: %q", i, displayWidth(row), width, row)
		}
	}
	joined := strings.Join(dangoBlock[:], "\n")
	for _, ch := range []string{"░", "▒", "▓", "█"} {
		if !strings.Contains(joined, ch) {
			t.Fatalf("missing %s in\n%s", ch, joined)
		}
	}
	if strings.Contains(joined, "_") || strings.Contains(joined, "|") {
		t.Fatalf("not a figlet:\n%s", joined)
	}
}

func TestSplashPaintsBeforeFetchAndDies(t *testing.T) {
	fetches := 0
	m := New(Options{
		Repo:   "archetype-labs/app",
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			fetches++
			return []domain.Stack{{
				ID:  "s",
				PRs: []domain.PullRequest{{Number: 1, Title: "landed layer"}},
			}}, nil
		},
	})
	if fetches != 0 {
		t.Fatalf("constructor fetched %d times", fetches)
	}
	if !m.splash() {
		t.Fatal("live open is a splash until the list exists")
	}
	raw := m.View()
	frame := stripANSI(raw)
	if fetches != 0 {
		t.Fatalf("View fetched %d times", fetches)
	}
	assertSplashFrame(t, frame, "fetching archetype-labs/app")
	if strings.Contains(frame, "YOINKS") {
		t.Fatalf("no joke words:\n%s", frame)
	}
	mr, mg, mb := rgb(domain.Color("meta"))
	pr, pg, pb := rgb(domain.Color("paper"))
	if !strings.Contains(raw, fmt.Sprintf("38;2;%d;%d;%d", mr, mg, mb)) {
		t.Fatal("fetching line is meta")
	}
	if !strings.Contains(raw, fmt.Sprintf("38;2;%d;%d;%d", pr, pg, pb)) {
		t.Fatal("block letters are paper")
	}

	next, _ := applyFetch(m)
	if fetches != 1 {
		t.Fatalf("Init fetches once, got %d", fetches)
	}
	if next.splash() {
		t.Fatal("splash dies once fetch returns")
	}
	listed := stripANSI(next.View())
	if strings.Contains(listed, "fetching archetype-labs/app") || strings.Contains(listed, dangoBlock[0]) {
		t.Fatalf("splash leftover:\n%s", listed)
	}
	if !strings.Contains(listed, "landed layer") {
		t.Fatalf("list:\n%s", listed)
	}
}

func TestFailedFetchStaysOnSplashWithArgv(t *testing.T) {
	argv := live.FormatGHArgv(append([]string{"pr", "list", "--repo", "archetype-labs/app", "--state", "open", "--limit", "100", "--json"}, strings.Join([]string{
		"number", "title", "url", "headRefName", "baseRefName", "author", "labels", "isDraft", "state",
	}, ",")))
	err502 := errors.New(argv + ": HTTP 502: Bad Gateway")
	var copied string
	old := copyText
	copyText = func(s string) error { copied = s; return nil }
	t.Cleanup(func() { copyText = old })

	m := New(Options{
		Repo:   "archetype-labs/app",
		Width:  80,
		Height: 24,
		Fetch:  func(string) ([]domain.Stack, error) { return nil, err502 },
	})
	assertSplashFrame(t, stripANSI(m.View()), "fetching archetype-labs/app")
	m, extra := applyFetch(m)
	if extra != nil {
		t.Fatal("failed fetch stays on the splash; do not quit")
	}
	if !m.splash() {
		t.Fatal("failed fetch must stay on the splash")
	}
	frame := stripANSI(m.View())
	if strings.Contains(frame, "Could not fetch pull requests.") {
		t.Fatalf("no empty-list error state:\n%s", frame)
	}
	if strings.Contains(frame, "No open stacks") {
		t.Fatalf("no sloppy empty list:\n%s", frame)
	}
	if !strings.Contains(frame, "502") || !strings.Contains(frame, "pr list") {
		t.Fatalf("loading line becomes the error:\n%s", frame)
	}
	if !strings.Contains(frame, "archetype-labs/app") {
		t.Fatalf("error names the real repo:\n%s", frame)
	}
	if strings.Contains(frame, "YOINKS") {
		t.Fatalf("no joke words:\n%s", frame)
	}
	if strings.Contains(frame, "●-●-● DANGO") {
		t.Fatalf("error splash is not the list header:\n%s", frame)
	}
	if !strings.Contains(frame, "[ . ]") {
		t.Fatalf("error splash keeps copy:\n%s", frame)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(Model)
	if copied != err502.Error() {
		t.Fatalf("dot copies the whole error block, got %q", copied)
	}
	if !strings.Contains(copied, "--json") || !strings.Contains(copied, "isDraft") {
		t.Fatalf("copied error must include exact argv: %q", copied)
	}
	if cmd == nil {
		t.Fatal("copy toast should clear")
	}
	if !strings.Contains(stripANSI(m.View()), "copied") {
		t.Fatalf("footer must flash copied:\n%s", stripANSI(m.View()))
	}
	if !m.splash() {
		t.Fatal("copy must not leave the splash")
	}
}

func TestSplashDotCopiesArgvWhileFetching(t *testing.T) {
	want := live.FormatGHArgv(live.PRListArgs("archetype-labs/app"))
	live.LastGHArgv = nil
	var copied string
	old := copyText
	copyText = func(s string) error { copied = s; return nil }
	t.Cleanup(func() { copyText = old })

	blocked := make(chan struct{})
	m := New(Options{
		Repo:   "archetype-labs/app",
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			<-blocked
			return nil, errors.New("should not finish")
		},
	})
	if !m.splash() || m.showError() {
		t.Fatal("still fetching: splash, not showError")
	}
	assertSplashFrame(t, stripANSI(m.View()), "fetching archetype-labs/app")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(Model)
	if copied != want {
		t.Fatalf("while fetching, copy exact argv\ngot  %q\nwant %q", copied, want)
	}
	if !strings.Contains(copied, "archetype-labs/app") || !strings.Contains(copied, "--json") {
		t.Fatalf("argv %q", copied)
	}
	if cmd == nil {
		t.Fatal("copy toast should clear")
	}
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "copied") {
		t.Fatalf("footer must flash copied while fetching:\n%s", frame)
	}
	if !m.splash() {
		t.Fatal("dot must not leave the splash")
	}
	if strings.Contains(frame, "No open stacks") {
		t.Fatalf("no empty list:\n%s", frame)
	}
	close(blocked)
}

func TestSplashDotKeepsArgvOnLineWhenCopyFails(t *testing.T) {
	want := live.FormatGHArgv(live.PRListArgs("archetype-labs/app"))
	live.LastGHArgv = nil
	old := copyText
	copyText = func(string) error { return errors.New("osc52 and pbcopy failed") }
	t.Cleanup(func() { copyText = old })

	m := New(Options{
		Repo:   "archetype-labs/app",
		Width:  80,
		Height: 24,
		Fetch:  func(string) ([]domain.Stack, error) { return nil, errors.New("unused") },
	})
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(Model)
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "copied") {
		t.Fatalf("toast still flashes:\n%s", frame)
	}
	if !strings.Contains(frame, "pr list") || !strings.Contains(frame, "archetype-labs/app") {
		t.Fatalf("failed copy keeps argv on the splash line:\n%s", frame)
	}
	if !strings.Contains(frame, want) && !strings.Contains(frame, "--json") {
		t.Fatalf("argv visible:\n%s", frame)
	}
	if !m.splash() {
		t.Fatal("stay on splash")
	}
}

func rgb(hex string) (r, g, b int) {
	r, g, b, _ = domain.ParseRGB(hex)
	return r, g, b
}

func assertSplashFrame(t *testing.T, frame, loading string) {
	t.Helper()
	for _, row := range dangoBlock {
		if !strings.Contains(frame, row) {
			t.Fatalf("missing %q:\n%s", row, frame)
		}
	}
	if strings.Count(frame, "●-●-●") != 1 {
		t.Fatalf("splash needs one ●-●-● under the letters:\n%s", frame)
	}
	if !strings.Contains(frame, loading) {
		t.Fatalf("missing %q:\n%s", loading, frame)
	}
	if strings.Contains(frame, "YOINKS") || strings.Contains(strings.ToLower(frame), "yoinks") {
		t.Fatalf("no joke words:\n%s", frame)
	}
	if strings.Contains(frame, "●-●-● DANGO") {
		t.Fatalf("splash is not the list header:\n%s", frame)
	}
	if strings.Contains(frame, "stacks /") || strings.Contains(frame, "last fetched") {
		t.Fatalf("splash is not list chrome:\n%s", frame)
	}
}
