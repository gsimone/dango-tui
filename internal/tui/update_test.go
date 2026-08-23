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

func TestSummariesAreAsyncAndLandInPlace(t *testing.T) {
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
	if len(m.stacks) != 1 || m.stacks[0].Name != "alpha layer" || m.stacks[0].Summary != "" {
		t.Fatalf("first paint is the gh name, no generated title: %+v", m.stacks)
	}
	first := stripANSI(m.View())
	if !strings.Contains(strings.Join(listNames(first), "\n"), "alpha layer") {
		t.Fatalf("first paint keeps the gh name:\n%s", first)
	}
	if strings.Contains(first, "alpha layer and beta layer") {
		t.Fatalf("first paint must not wait on a generated title:\n%s", first)
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("provider must kick a summary cmd after first paint")
	}
	if m.Fetching || strings.Contains(stripANSI(m.View()), "⠋") {
		t.Fatal("title wait must not be a spinner")
	}
	if New(Options{Repo: "owner/name", Width: 80, Height: 24, Fetch: fetch}).Init() != nil {
		t.Fatal("missing provider must not start summaries")
	}

	next, extra := m.Update(summaryDoneMsg{token: m.fetchSeq, id: "s"})
	m = next.(Model)
	if extra != nil || m.stacks[0].Name != "alpha layer" {
		t.Fatalf("empty generated title keeps the gh name: %+v", m.stacks[0])
	}

	next, extra = m.Update(summaryDoneMsg{token: m.fetchSeq, id: "s", description: "later"})
	m = next.(Model)
	if extra != nil || m.stacks[0].Name != "alpha layer" || m.stacks[0].Description != "later" {
		t.Fatalf("description is inspector-only: %+v", m.stacks[0])
	}
	if !strings.Contains(stripANSI(m.View()), "later") {
		t.Fatal("description fills the inspector pane")
	}

	next, extra = m.Update(summaryDoneMsg{token: m.fetchSeq, id: "s", title: "from hook", description: "later"})
	if extra != nil {
		t.Fatal("summary fill is not a new fetch")
	}
	m = next.(Model)
	if m.stacks[0].Name != "from hook" || m.stacks[0].Description != "later" {
		t.Fatalf("cmd land should swap the title in place: %+v", m.stacks[0])
	}

	next, _ = m.Update(summaryDoneMsg{token: m.fetchSeq + 1, id: "s", title: "stale"})
	m = next.(Model)
	if m.stacks[0].Name != "from hook" {
		t.Fatal("stale summary must not overwrite")
	}

	live := New(Options{
		Repo:     "owner/name",
		Provider: summary.ParseProvider("codex@luna.medium"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
	})
	landed := applyCmd(live, live.Init())
	if landed.stacks[0].Name != "alpha layer and beta layer" {
		t.Fatalf("real Run must land the generated title: %+v", landed.stacks[0])
	}
	if landed.stacks[0].Description == "" {
		t.Fatal("real Run must land an inspector description")
	}
	after := stripANSI(landed.View())
	if !strings.Contains(after, landed.stacks[0].Description) {
		t.Fatalf("description fills the inspector:\n%s", after)
	}
	if strings.Contains(after, "⠋") {
		t.Fatalf("in-place fill is not a spinner:\n%s", after)
	}
	if len(listNames(after)) != len(listNames(first)) {
		t.Fatalf("title swap must not rebuild the table: before %v after %v", listNames(first), listNames(after))
	}

	mocked := New(Options{
		Repo:     "owner/name",
		Provider: summary.ParseProvider("mock@test"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
		Summarize: func(job summary.Job) summary.Result {
			if job.Provider.Raw != "mock@test" || job.ID != "s" {
				t.Fatalf("job %+v", job)
			}
			return summary.Result{ID: job.ID, Title: "mocked title", Description: "mocked inspector copy"}
		},
	})
	if strings.Contains(stripANSI(mocked.View()), "mocked title") {
		t.Fatal("mocked provider must not block first paint")
	}
	mocked = applyCmd(mocked, mocked.Init())
	if mocked.stacks[0].Name != "mocked title" || mocked.stacks[0].Description != "mocked inspector copy" {
		t.Fatalf("mocked provider must run: %+v", mocked.stacks[0])
	}
	mockedFrame := stripANSI(mocked.View())
	if !strings.Contains(strings.Join(listNames(mockedFrame), "\n"), "mocked title") {
		t.Fatalf("mocked title replaces the list name:\n%s", mockedFrame)
	}
	if !strings.Contains(mockedFrame, "mocked inspector copy") {
		t.Fatalf("mocked description fills the inspector:\n%s", mockedFrame)
	}
}

func applyCmd(m Model, cmd tea.Cmd) Model {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, c := range batch {
			m = applyCmd(m, c)
		}
		return m
	}
	next, nextCmd := m.Update(msg)
	return applyCmd(next.(Model), nextCmd)
}

func listNames(frame string) []string {
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

func TestInspectorStatusInkIsValueOnly(t *testing.T) {
	auth := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[0]
	head := auth.PRs[len(auth.PRs)-1]
	facts := inspectorFacts(head)
	if facts[0].label != "status" || facts[0].value != "CI failing" {
		t.Fatalf("status fact: %+v", facts[0])
	}
	if facts[0].fg != inspectorStatusColor(head) || facts[0].fg != domain.Color("ciFailure") {
		t.Fatalf("status value must use status color, got %s", facts[0].fg)
	}
	paper := domain.Color("paper")
	for _, fact := range facts[1:] {
		switch fact.label {
		case "labels", "author":
			if len(fact.parts) == 0 {
				t.Fatalf("%s needs painted parts", fact.label)
			}
		default:
			if fact.fg != paper {
				t.Fatalf("%s must stay paper, got %s", fact.label, fact.fg)
			}
		}
	}

	base := New(Options{StoryID: "mixed", Width: 120, Height: 30}).Stacks()[0].PRs[0]
	facts = inspectorFacts(base)
	var labels, author inspectorFact
	for _, fact := range facts {
		switch fact.label {
		case "labels":
			labels = fact
		case "author":
			author = fact
		}
	}
	if labels.value != "bug auth" || len(labels.parts) < 3 {
		t.Fatalf("labels fact: %+v", labels)
	}
	if labels.parts[0].text != "bug" || labels.parts[0].fg != "#d73a4a" {
		t.Fatalf("bug label ink: %+v", labels.parts[0])
	}
	if labels.parts[2].text != "auth" || labels.parts[2].fg != "#0e8a16" {
		t.Fatalf("auth label ink: %+v", labels.parts[2])
	}
	if author.value != "● gianni" || author.parts[0].text != "●" || author.parts[0].fg != domain.LoginColor("gianni") {
		t.Fatalf("author fact: %+v", author)
	}

	blocked := New(Options{StoryID: "changes-requested", Width: 120, Height: 30}).Stacks()[0].PRs[0]
	if got := inspectorStatusColor(blocked); got != domain.Color("reviewBlocked") {
		t.Fatalf("blocked status value %s", got)
	}
	merged := auth.PRs[0]
	if inspectorStatusColor(merged) != domain.Color("merged") {
		t.Fatalf("merged status value %s", inspectorStatusColor(merged))
	}
}

func TestDotCopiesBranchToast(t *testing.T) {
	var copied string
	old := copyText
	copyText = func(s string) { copied = s }
	t.Cleanup(func() { copyText = old })

	before := gitHEAD(t)
	m := New(Options{StoryID: "mixed", Width: 80, Height: 24})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(Model)
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "copied gm/stacks-184") {
		t.Fatalf("toast:\n%s", frame)
	}
	if strings.Contains(frame, "Copied ") {
		t.Fatalf("toast is lowercase copied, not Copied:\n%s", frame)
	}
	if strings.Contains(frame, "Checked out") {
		t.Fatalf("copy must not checkout:\n%s", frame)
	}
	if strings.Contains(frame, "[ ? ] close") {
		t.Fatalf("copy must not open help:\n%s", frame)
	}
	if strings.Contains(frame, "[ p ]") {
		t.Fatalf("no picker:\n%s", frame)
	}
	if copied != "gm/stacks-184" {
		t.Fatalf("copied %q", copied)
	}
	if after := gitHEAD(t); after != before {
		t.Fatalf("changed git HEAD: %s -> %s", before, after)
	}
	if cmd == nil {
		t.Fatal("toast should clear")
	}
	next, _ = m.Update(clearFeedbackMsg{token: m.feedbackSeq})
	m = next.(Model)
	cleared := stripANSI(m.View())
	if strings.Contains(cleared, "copied gm/stacks-184") {
		t.Fatalf("toast should clear:\n%s", cleared)
	}
	if !strings.Contains(cleared, "[ ↑↓ ] stack") || !strings.Contains(cleared, "[ o ] open") || !strings.Contains(cleared, "[ . ] copy") {
		t.Fatalf("footer should return to the key legend:\n%s", cleared)
	}
	if strings.Contains(cleared, "[ enter ]") {
		t.Fatalf("enter must leave the footer:\n%s", cleared)
	}
}

func TestCommaDoesNotCopy(t *testing.T) {
	var copied string
	old := copyText
	copyText = func(s string) { copied = s }
	t.Cleanup(func() { copyText = old })

	m := New(Options{StoryID: "mixed", Width: 80, Height: 24})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(",")})
	m = next.(Model)
	frame := stripANSI(m.View())
	if copied != "" {
		t.Fatalf(", is unbound, copied %q", copied)
	}
	if cmd != nil {
		t.Fatal(", must not start a toast clear")
	}
	if strings.Contains(frame, "copied ") {
		t.Fatalf(", must not toast:\n%s", frame)
	}
	if !strings.Contains(frame, "[ . ] copy") {
		t.Fatalf("footer stays [ . ] copy:\n%s", frame)
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
