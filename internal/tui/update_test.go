package tui

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/summary"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*(?:\x07|\x1b\\)`)

func TestRelativeFetched(t *testing.T) {
	at := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	if got := relativeFetched(at, at); got != "last fetched just now" {
		t.Fatalf("just now: %q", got)
	}
	if got := relativeFetched(at, at.Add(time.Minute)); got != "last fetched 1 min ago" {
		t.Fatalf("one min: %q", got)
	}
	if got := relativeFetched(at, at.Add(2*time.Minute)); got != "last fetched 2 mins ago" {
		t.Fatalf("two mins: %q", got)
	}
}

func TestSummariesAreAsyncAndStubbed(t *testing.T) {
	fetch := func(string) ([]domain.Stack, error) {
		return []domain.Stack{{
			ID:  "s",
			PRs: []domain.PullRequest{{Number: 1, Title: "alpha layer"}, {Number: 2, Title: "beta layer"}},
		}}, nil
	}
	m := New(Options{
		Repo:     "owner/name",
		Provider: summary.ParseProvider("codex@luna.medium"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
	})
	if len(m.stacks) != 1 || m.stacks[0].Name != "" || m.stacks[0].Summary != "" {
		t.Fatalf("first model must not wait on the summarizer: %+v", m.stacks)
	}
	if cmd := m.Init(); cmd == nil {
		t.Fatal("provider must kick a summary cmd after first paint")
	}
	if cmd := New(Options{Repo: "owner/name", Width: 80, Height: 24, Fetch: fetch}).Init(); cmd != nil {
		t.Fatal("missing provider must not start summaries")
	}

	next, cmd := m.Update(summaryDoneMsg{token: m.fetchSeq, id: "s", title: "from hook", description: "later"})
	if cmd != nil {
		t.Fatal("summary fill is not a new fetch")
	}
	m = next.(Model)
	if m.stacks[0].Name != "from hook" || m.stacks[0].Description != "later" {
		t.Fatalf("cmd land should fill title/description: %+v", m.stacks[0])
	}

	next, _ = m.Update(summaryDoneMsg{token: m.fetchSeq + 1, id: "s", title: "stale"})
	m = next.(Model)
	if m.stacks[0].Name != "from hook" {
		t.Fatal("stale summary must not overwrite")
	}
}

func TestStatusWordsKeepStatusColor(t *testing.T) {
	auth := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[0]
	if got := stackHealthColor(auth); got != domain.Color("ciFailure") {
		t.Fatalf("auth head should be ciFailure, got %s", got)
	}
	composer := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[1]
	if got := stackHealthColor(composer); got != domain.Color("queued") {
		t.Fatalf("composer head should be queued, got %s", got)
	}
}

func TestLayerBallsUseStatusTokens(t *testing.T) {
	auth := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[0]
	want := []string{domain.Color("merged"), domain.Color("ready"), domain.Color("ciFailure")}
	if len(auth.PRs) != 3 {
		t.Fatalf("auth should have 3 layers, got %d", len(auth.PRs))
	}
	for i, pr := range auth.PRs {
		if got := layerBallInk(pr); got != want[i] {
			t.Fatalf("layer %d ink %s, want status %s", i, got, want[i])
		}
		if domain.IsLogoToken(domain.StateColorToken(domain.GetDisplayState(pr))) {
			t.Fatalf("layer %d must not use a logo hue token", i)
		}
	}
}

func TestRowDangerInkMatchesListStatus(t *testing.T) {
	auth := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[0]
	ink, red := rowDangerInk(auth)
	if !red || ink != stackHealthColor(auth) || ink != domain.Color("ciFailure") {
		t.Fatalf("auth row is red; card values must use the list token, got %s red=%v", ink, red)
	}
	composer := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[1]
	if _, red := rowDangerInk(composer); red {
		t.Fatal("queued row is not a red status")
	}
	blocked := New(Options{StoryID: "changes-requested", Width: 120, Height: 30}).Stacks()[0]
	ink, red = rowDangerInk(blocked)
	if !red || ink != domain.Color("reviewBlocked") {
		t.Fatalf("blocked row must use reviewBlocked, got %s red=%v", ink, red)
	}
}

func TestDotCopiesBranchAndDoesNotCheckout(t *testing.T) {
	var copied string
	old := copyText
	copyText = func(s string) { copied = s }
	t.Cleanup(func() { copyText = old })

	before := gitHEAD(t)
	m := New(Options{StoryID: "mixed", Width: 80, Height: 24})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(Model)
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "Copied gm/stacks-184") {
		t.Fatalf("expected quiet copy confirmation:\n%s", frame)
	}
	if strings.Contains(frame, "Checked out") {
		t.Fatalf("dot must not checkout:\n%s", frame)
	}
	if copied != "gm/stacks-184" {
		t.Fatalf("copied %q, want gm/stacks-184", copied)
	}
	if after := gitHEAD(t); after != before {
		t.Fatalf("dot changed git HEAD: %s -> %s", before, after)
	}
	if cmd == nil {
		t.Fatal("copy confirmation should clear")
	}

	next, _ = m.Update(clearFeedbackMsg{token: m.feedbackSeq})
	m = next.(Model)
	cleared := stripANSI(m.View())
	if strings.Contains(cleared, "Copied gm/stacks-184") {
		t.Fatalf("copy confirmation should clear:\n%s", cleared)
	}
	if !strings.Contains(cleared, "[ ↑↓ ] stack") || !strings.Contains(cleared, "[ o ] open") || !strings.Contains(cleared, "[ . ] copy") {
		t.Fatalf("footer should return to the key legend:\n%s", cleared)
	}
	if strings.Contains(cleared, "[ enter ]") {
		t.Fatalf("enter must leave the footer:\n%s", cleared)
	}
}

func TestOpenRestoresFooterAfterResult(t *testing.T) {
	m := New(Options{StoryID: "mixed", Width: 80, Height: 24})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = next.(Model)
	opening := stripANSI(m.View())
	if !strings.Contains(opening, "Opening https://github.com/example/stacks/pull/184") {
		t.Fatalf("open should show a brief status:\n%s", opening)
	}
	if cmd == nil {
		t.Fatal("open should start a command")
	}

	next, _ = m.Update(openResultMsg{url: "https://github.com/example/stacks/pull/184"})
	m = next.(Model)
	okFrame := stripANSI(m.View())
	if strings.Contains(okFrame, "Opening ") {
		t.Fatalf("success must not leave Opening stuck:\n%s", okFrame)
	}
	if !strings.Contains(okFrame, "[ ↑↓ ] stack") || !strings.Contains(okFrame, "[ o ] open") || !strings.Contains(okFrame, "[ . ] copy") {
		t.Fatalf("success should restore the key legend:\n%s", okFrame)
	}
	if strings.Contains(okFrame, "[ enter ]") {
		t.Fatalf("enter must leave the footer:\n%s", okFrame)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	m = next.(Model)
	next, _ = m.Update(openResultMsg{
		url: "https://github.com/example/stacks/pull/184",
		err: errOpenFailed,
	})
	m = next.(Model)
	failFrame := stripANSI(m.View())
	if strings.Contains(failFrame, "Opening ") || strings.Contains(failFrame, "Could not open") {
		t.Fatalf("failure must not lock the footer:\n%s", failFrame)
	}
	if !strings.Contains(failFrame, "[ ↑↓ ] stack") || !strings.Contains(failFrame, "[ o ] open") || !strings.Contains(failFrame, "[ . ] copy") {
		t.Fatalf("failure should restore the key legend:\n%s", failFrame)
	}
	if strings.Contains(failFrame, "[ enter ]") {
		t.Fatalf("enter must leave the footer:\n%s", failFrame)
	}
}

type boomError struct{}

func (boomError) Error() string { return "boom" }

var errOpenFailed error = boomError{}

func gitHEAD(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("git rev-parse HEAD: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
