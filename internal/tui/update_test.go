package tui

import (
	"os/exec"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*(?:\x07|\x1b\\)`)

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
	if !strings.Contains(cleared, "↑↓ stack  ←→ layer  enter checkout  o open  . copy") {
		t.Fatalf("footer should return to the key legend:\n%s", cleared)
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
	if !strings.Contains(okFrame, "↑↓ stack  ←→ layer  enter checkout  o open  . copy") {
		t.Fatalf("success should restore the key legend:\n%s", okFrame)
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
	if !strings.Contains(failFrame, "↑↓ stack  ←→ layer  enter checkout  o open  . copy") {
		t.Fatalf("failure should restore the key legend:\n%s", failFrame)
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
