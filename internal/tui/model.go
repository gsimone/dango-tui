package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
)

type Options struct {
	StoryID string
	Width   int
	Height  int
}

type Model struct {
	Width      int
	Height     int
	StoryIndex int
	State      app.State
	Help       bool
	quitting   bool
}

func New(opts Options) Model {
	width, height := opts.Width, opts.Height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	storyID := opts.StoryID
	if storyID == "" {
		storyID = "mixed"
	}
	idx := 0
	for i, story := range data.FixtureStories {
		if story.ID == storyID {
			idx = i
			break
		}
	}
	return Model{
		Width:      width,
		Height:     height,
		StoryIndex: idx,
		State:      app.InitialState(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Story() data.FixtureStory {
	if m.StoryIndex < 0 || m.StoryIndex >= len(data.FixtureStories) {
		return data.StoryByID("mixed")
	}
	return data.FixtureStories[m.StoryIndex]
}

func (m Model) Stacks() []domain.Stack {
	return app.FilterStacks(m.Story().Stacks, m.State.Query)
}

func (m Model) SelectedStack() (domain.Stack, bool) {
	stacks := m.Stacks()
	sel := app.ClampSelection(m.State.Selection, stacks)
	if len(stacks) == 0 || sel.StackIndex >= len(stacks) {
		return domain.Stack{}, false
	}
	return stacks[sel.StackIndex], true
}

func (m Model) SelectedPR() (domain.PullRequest, bool) {
	stack, ok := m.SelectedStack()
	if !ok {
		return domain.PullRequest{}, false
	}
	sel := app.ClampSelection(m.State.Selection, m.Stacks())
	if sel.PRIndex < 0 || sel.PRIndex >= len(stack.PRs) {
		return domain.PullRequest{}, false
	}
	return stack.PRs[sel.PRIndex], true
}

func (m *Model) choose(next app.Selection, reveal bool) {
	m.State.Selection = app.ClampSelection(next, m.Stacks())
	if reveal {
		m.State.CardVisible = true
	}
}

func (m *Model) checkout(pr domain.PullRequest) {
	m.State.Feedback = "Checked out " + pr.Branch + " · fixture simulation"
	m.State.CardVisible = true
}

func (m *Model) clamp() {
	m.State.Selection = app.ClampSelection(m.State.Selection, m.Stacks())
}

func (m Model) sourceState() string {
	switch m.Story().CacheState {
	case data.CacheStale:
		return "fixture cache · stale (simulated)"
	case data.CacheError:
		return "fixture refresh failed · no cached stacks"
	default:
		return "fixture cache · current · no network"
	}
}

func (m Model) emptyMessage() string {
	if strings.TrimSpace(m.State.Query) != "" {
		return "No fixture stack matches this filter."
	}
	if m.Story().CacheState == data.CacheError {
		return "Refresh failed in this fixture. No cached stacks are available."
	}
	return "No open stacks in this fixture repository."
}

func (m Model) stackCount() int {
	return len(m.Story().Stacks)
}

func (m Model) layerCount() int {
	n := 0
	for _, stack := range m.Story().Stacks {
		n += len(stack.PRs)
	}
	return n
}

func (m Model) title() string {
	return "STACKS"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func deleteLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
