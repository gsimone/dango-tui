package tui

import (
	"errors"
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

func TestStackListNameKeepsShortTitle(t *testing.T) {
	gh := "LEV-182: Bound hosts to the session"
	stack := domain.Stack{
		Name: "LEV-182",
		PRs:  []domain.PullRequest{{Title: gh}, {Title: "head"}},
	}
	if got := stackListName(stack); got != "LEV-182" {
		t.Fatalf("list keeps the ticket, got %q", got)
	}
	named := stack
	named.Name = "from the title agent"
	if got := stackListName(named); got != "from the title agent" {
		t.Fatalf("landed provider title still swaps in place, got %q", got)
	}
	stamped := domain.Stack{Name: gh, PRs: []domain.PullRequest{{Title: gh}, {Title: "head"}}}
	if got := stackListName(stamped); got != "LEV-182" {
		t.Fatalf("full sentence belongs in the pane, got %q", got)
	}
	blank := domain.Stack{PRs: []domain.PullRequest{{Title: gh}, {Title: "head"}}}
	if got := stackListName(blank); got != "LEV-182" {
		t.Fatalf("empty name uses the short gh title, got %q", got)
	}
}

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

func applyFetch(m Model) (Model, tea.Cmd) {
	cmd := m.Init()
	if cmd == nil {
		return m, nil
	}
	next, extra := m.Update(cmd())
	return next.(Model), extra
}

func TestSummariesAreAsyncAndLandInPlace(t *testing.T) {
	fetch := func(string) ([]domain.Stack, error) {
		return []domain.Stack{{
			ID:  "s",
			PRs: []domain.PullRequest{{Number: 1, Title: "alpha layer"}, {Number: 2, Title: "beta layer"}},
		}}, nil
	}
	m := New(Options{
		Repo:     "archetype-labs/app",
		Provider: summary.ParseProvider("codex@luna.medium"),
		Width:    80,
		Height:   24,
		Fetch:    fetch,
	})
	if len(m.stacks) != 0 {
		t.Fatal("constructor must not wait on gh")
	}
	if !strings.Contains(stripANSI(m.View()), "fetching archetype-labs/app") {
		t.Fatalf("first frame before gh:\n%s", stripANSI(m.View()))
	}
	var sumCmd tea.Cmd
	m, sumCmd = applyFetch(m)
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
	if sumCmd == nil {
		t.Fatal("provider must kick a summary cmd after first paint")
	}
	if m.Fetching || strings.Contains(stripANSI(m.View()), "⠋") {
		t.Fatal("title wait must not be a spinner")
	}
	_, extra := applyFetch(New(Options{Repo: "archetype-labs/app", Width: 80, Height: 24, Fetch: fetch}))
	if extra != nil {
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

func TestSummaryDonePaneDescriptionIsNotGhTitle(t *testing.T) {
	gh := "LEV-182: Bound hosts to the session"
	fetch := func(string) ([]domain.Stack, error) {
		return []domain.Stack{{
			ID:   "s",
			Name: gh,
			PRs: []domain.PullRequest{
				{Number: 182, Title: gh, Body: "<!-- CURSOR_AGENT_PR_BODY_BEGIN -->\nPin each bound host to the worker.\n"},
				{Number: 183, Title: "head layer"},
			},
		}}, nil
	}
	m := New(Options{
		Repo:     "gsimone/leva-2",
		Provider: summary.ParseProvider("local"),
		Width:    120,
		Height:   30,
		Fetch:    fetch,
	})
	m, _ = applyFetch(m)
	before := stripANSI(m.View())
	if !strings.Contains(strings.Join(listNames(before), "\n"), "LEV-182") {
		t.Fatalf("first paint is the gh name:\n%s", before)
	}
	if strings.Contains(before, "Pin each bound host") || strings.Contains(before, "Covers ") || strings.Contains(before, "CURSOR_AGENT") {
		t.Fatalf("first paint must not wait on generated copy:\n%s", before)
	}
	statusAt := factRow(before, "status")

	res := summary.Run(summary.Job{
		Provider: summary.ParseProvider("local"),
		ID:       "s",
		Stack:    m.stacks[0],
	})
	if res.Description == gh || strings.Contains(res.Description, "CURSOR_AGENT") || strings.Contains(res.Description, "Pin each") || strings.HasPrefix(res.Description, "Covers ") {
		t.Fatalf("Run must not paste body or wrap Covers: %q", res.Description)
	}

	next, cmd := m.Update(summaryDoneMsg{token: m.fetchSeq, id: "s", title: res.Title, description: res.Description})
	if cmd != nil {
		t.Fatal("land is not a fetch")
	}
	m = next.(Model)
	after := stripANSI(m.View())
	if m.stacks[0].Description == gh || strings.EqualFold(m.stacks[0].Description, gh) {
		t.Fatalf("landed description echoed gh title: %q", m.stacks[0].Description)
	}
	if strings.Contains(after, "CURSOR_AGENT") || strings.Contains(after, "<!--") || strings.Contains(after, "Pin each bound host") {
		t.Fatalf("raw body leaked into the pane:\n%s", after)
	}
	if m.stacks[0].Description != "" && !strings.Contains(after, m.stacks[0].Description) {
		t.Fatalf("pane must show the distinct clause:\n%s", after)
	}
	if strings.Contains(after, "Covers ") {
		t.Fatalf("do not invent a Covers wrapper:\n%s", after)
	}
	if after == before {
		t.Fatal("pane text must change after summaryDoneMsg")
	}
	if strings.Contains(after, "⠋") {
		t.Fatalf("no list spinner:\n%s", after)
	}
	if factRow(after, "status") != statusAt {
		t.Fatalf("fact rows moved: before %d after %d", statusAt, factRow(after, "status"))
	}
	if m.stacks[0].Name == gh && m.stacks[0].Description == gh {
		t.Fatal("land was a no-op")
	}
}

func TestDescriptionFillsInspectorInPlace(t *testing.T) {
	fetch := func(string) ([]domain.Stack, error) {
		return []domain.Stack{{
			ID:  "s",
			PRs: []domain.PullRequest{{Number: 1, Title: "alpha layer"}, {Number: 2, Title: "beta layer"}},
		}}, nil
	}
	for _, size := range []struct{ w, h int }{{80, 24}, {120, 30}} {
		m := New(Options{
			Repo:     "owner/name",
			Provider: summary.ParseProvider("codex@luna.medium"),
			Width:    size.w,
			Height:   size.h,
			Fetch:    fetch,
		})
		var sumCmd tea.Cmd
		m, sumCmd = applyFetch(m)
		before := stripANSI(m.View())
		if strings.Contains(before, "alpha layer and beta layer") {
			t.Fatalf("%dx%d first paint waited on a description:\n%s", size.w, size.h, before)
		}
		if strings.Contains(before, "⠋") {
			t.Fatalf("%dx%d first paint must not spin the list:\n%s", size.w, size.h, before)
		}
		statusAt := factRow(before, "status")
		ciAt := factRow(before, "ci")
		if statusAt < 0 || ciAt < 0 {
			t.Fatalf("%dx%d reserved pane missing facts:\n%s", size.w, size.h, before)
		}
		box := boxBounds(before)

		m = applyCmd(m, sumCmd)
		after := stripANSI(m.View())
		if m.stacks[0].Description == "" {
			t.Fatal("description must land")
		}
		if !strings.Contains(after, m.stacks[0].Description) && !strings.Contains(after, strings.Fields(m.stacks[0].Description)[0]) {
			t.Fatalf("%dx%d description must fill the inspector:\n%s", size.w, size.h, after)
		}
		if strings.Contains(after, "⠋") {
			t.Fatalf("%dx%d land must not spin the list:\n%s", size.w, size.h, after)
		}
		if size.w >= 100 {
			if factRow(after, "status") != statusAt || factRow(after, "ci") != ciAt {
				t.Fatalf("%dx%d fact rows moved (pane morph): before status=%d ci=%d after status=%d ci=%d\n%s",
					size.w, size.h, statusAt, ciAt, factRow(after, "status"), factRow(after, "ci"), after)
			}
			if boxBounds(after) != box {
				t.Fatalf("%dx%d pane morphed: before %v after %v", size.w, size.h, box, boxBounds(after))
			}
		}
		if len(listNames(after)) != len(listNames(before)) {
			t.Fatalf("%dx%d list rows changed: before %v after %v", size.w, size.h, listNames(before), listNames(after))
		}
	}
}

func factRow(frame, label string) int {
	for i, line := range strings.Split(frame, "\n") {
		part := line
		if idx := strings.Index(line, "│"); idx >= 0 {
			part = line[idx:]
		}
		if strings.Contains(part, label) && (strings.Contains(part, label+"    ") || strings.Contains(part, label+"   ")) {
			return i
		}
	}
	return -1
}

func boxBounds(frame string) [2]int {
	top, bot := -1, -1
	for i, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, "┌") {
			top = i
		}
		if strings.Contains(line, "└") {
			bot = i
		}
	}
	return [2]int{top, bot}
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

func TestDotCopiesFetchError(t *testing.T) {
	sha := "dddddddddddddddddddddddddddddddddddddddd"
	withVCS(t, sha)
	err502 := errors.New("gh pr list --repo archetype-labs/app --state open --limit 100 --json number,title,url,headRefName,baseRefName,author,labels,isDraft,state: HTTP 502: Bad Gateway")
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
	if !strings.Contains(stripANSI(m.View()), "fetching archetype-labs/app") {
		t.Fatalf("first frame before gh:\n%s", stripANSI(m.View()))
	}
	m, extra := applyFetch(m)
	if extra != nil {
		t.Fatal("failed fetch must not quit")
	}
	frame := stripANSI(m.View())
	if strings.Contains(frame, "Could not fetch pull requests.") || strings.Contains(frame, "No open stacks") {
		t.Fatalf("stay on splash, no empty-list error:\n%s", frame)
	}
	if !strings.Contains(frame, "502") {
		t.Fatalf("loading line becomes the error:\n%s", frame)
	}
	if !strings.Contains(frame, "pr list") || !strings.Contains(frame, "archetype-labs/app") {
		t.Fatalf("error block missing exact argv:\n%s", frame)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	m = next.(Model)
	if !strings.Contains(copied, err502.Error()) || !strings.Contains(copied, sha) {
		t.Fatalf("dot copies the error plus SHA, got %q", copied)
	}
	if cmd == nil {
		t.Fatal("copy toast should clear")
	}
	if !strings.Contains(stripANSI(m.View()), "copied") {
		t.Fatalf("footer must flash copied:\n%s", stripANSI(m.View()))
	}
	if m.View() == "" {
		t.Fatal("error must not quit the process")
	}
}

func TestDotCopiesBranchToast(t *testing.T) {
	var copied string
	old := copyText
	copyText = func(s string) error { copied = s; return nil }
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
	copyText = func(s string) error { copied = s; return nil }
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
