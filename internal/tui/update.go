package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/live"
)

type fetchDoneMsg struct {
	stacks []domain.Stack
	err    error
	at     time.Time
	token  int
	live   bool
	file   bool
	slug   string
}

type afterPaintMsg struct{}

type summaryDoneMsg struct {
	token       int
	id          string
	title       string
	description string
}

type ciDoneMsg struct {
	token  int
	stacks []domain.Stack
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.Width = msg.Width
		}
		if msg.Height > 0 {
			m.Height = msg.Height
		}
		return m, nil
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case fetchDoneMsg:
		if msg.live && msg.token != m.fetchSeq {
			return m, nil
		}
		m.Fetching = false
		m.State.Feedback = ""
		if msg.live {
			m.fetchedAt = msg.at
			if msg.at.IsZero() {
				m.fetchedAt = time.Now()
			}
			m.Fetched = relativeFetched(m.fetchedAt, time.Now())
			if msg.err != nil {
				m.fetchErr = msg.err
				m.cacheState = data.CacheError
			} else {
				m.fetchErr = nil
				m.stacks = live.KeepRealStacks(live.StampGhNames(msg.stacks))
				m.cacheState = data.CacheCurrent
				m.clamp()
				return m, m.afterFetch()
			}
		} else if msg.file || m.File {
			m.Fetched = "last fetched 2 mins ago"
			if msg.slug != "" {
				m.Repo = msg.slug
			}
			if msg.err != nil {
				m.fetchErr = msg.err
				m.cacheState = data.CacheError
			} else {
				m.fetchErr = nil
				m.stacks = msg.stacks
				m.cacheState = data.CacheCurrent
				m.clamp()
			}
		} else {
			m.Fetched = "last fetched 2 mins ago"
		}
		return m, nil
	case afterPaintMsg:
		return m, m.startSelectedSummary()
	case summaryDoneMsg:
		return m, m.applySummary(msg)
	case ciDoneMsg:
		if msg.token != m.fetchSeq {
			return m, nil
		}
		m.stacks = live.ApplyCI(m.stacks, msg.stacks)
		return m, nil
	case openResultMsg:
		if strings.HasPrefix(m.State.Feedback, "Opening ") {
			m.State.Feedback = ""
		}
		return m, nil
	case clearFeedbackMsg:
		if msg.token == m.feedbackSeq {
			m.State.Feedback = ""
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.State.Searching {
		switch msg.String() {
		case "esc", "escape":
			if m.State.Query != "" {
				m.State.Query = ""
			} else {
				m.State.Searching = false
			}
			m.State.CardVisible = true
			m.clamp()
			return m, nil
		case "backspace":
			m.State.Query = deleteLastRune(m.State.Query)
			m.clamp()
			return m, nil
		case "ctrl+c":
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				m.State.Query += string(msg.Runes)
				m.clamp()
			}
			return m, nil
		}
	}

	switch msg.String() {
	case "up":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirUp), true)
		return m, m.startSelectedSummary()
	case "down":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirDown), true)
		return m, m.startSelectedSummary()
	case "left":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirLeft), true)
	case "right":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirRight), true)
	case "home":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirHome), true)
		return m, m.startSelectedSummary()
	case "end":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirEnd), true)
		return m, m.startSelectedSummary()
	case "o":
		if pr, ok := m.SelectedPR(); ok {
			return m, m.open(pr)
		}
	case ".":
		if m.splash() {
			return m, m.copySplash()
		}
		if m.showError() {
			return m, m.copyError()
		}
		if pr, ok := m.SelectedPR(); ok {
			return m, m.copyBranch(pr)
		}
	case "a":
		m.State.Feedback = "add · not wired"
	case "r":
		return m.refresh()
	case "/":
		m.Help = false
		m.State.Query = ""
		m.State.Searching = true
		m.State.CardVisible = false
	case "?":
		m.Help = !m.Help
	case "esc", "escape":
		if m.waiting() || m.showError() {
			m.quitting = true
			return m, tea.Quit
		}
		if m.Help {
			m.Help = false
		} else {
			m.State.CardVisible = false
		}
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "ctrl+c":
		return m, nil
	}
	return m, nil
}

func (m *Model) applySummary(msg summaryDoneMsg) tea.Cmd {
	if msg.token != m.fetchSeq {
		return nil
	}
	m.summaryBusy = false
	if m.summaryDone == nil {
		m.summaryDone = map[string]bool{}
	}
	m.summaryDone[msg.id] = true
	title := strings.TrimSpace(msg.title)
	desc := strings.TrimSpace(msg.description)
	var toast tea.Cmd
	if title != "" || desc != "" || strings.TrimSpace(m.Describe) != "" {
		idx := m.summaryStackIndex(msg.id)
		if idx >= 0 {
			if title != "" {
				m.stacks[idx].Name = title
				m.stacks[idx].Summary = title
			}
			if desc != "" {
				m.stacks[idx].Description = desc
				m.State.Feedback = "described"
				toast = m.clearFeedback()
			} else if strings.TrimSpace(m.Describe) != "" {
				m.stacks[idx].Description = ""
			}
			if m.stacks[idx].ID == "" && msg.id != "" {
				m.stacks[idx].ID = msg.id
			}
			m.summaryDone[m.stacks[idx].ID] = true
		}
	}
	return tea.Batch(toast, m.startSelectedSummary())
}

// summaryStackIndex is the live row a describe result belongs to.
// Matching ID wins. A miss still writes the selected stack — live
// grouping can stamp stack-N / gh-stack-N after the job captured a
// different id on the Update value-receiver copy.
func (m *Model) summaryStackIndex(id string) int {
	if id != "" {
		for i := range m.stacks {
			if m.stacks[i].ID == id {
				return i
			}
		}
	}
	stack, ok := m.SelectedStack()
	if !ok {
		return -1
	}
	for i := range m.stacks {
		if stackRefEquals(m.stacks[i], stack) {
			return i
		}
	}
	sel := m.State.Selection.StackIndex
	if sel >= 0 && sel < len(m.stacks) {
		return sel
	}
	return -1
}

func (m Model) refresh() (tea.Model, tea.Cmd) {
	m.Fetching = true
	m.State.Feedback = ""
	if m.File {
		path := m.file
		return m, func() tea.Msg {
			repo, stacks, err := data.LoadStacks(path)
			return fetchDoneMsg{stacks: stacks, err: err, at: time.Now(), token: 0, live: false, file: true, slug: repo}
		}
	}
	if !m.Live {
		return m, tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg { return fetchDoneMsg{} })
	}
	m.fetchSeq++
	m.summaryBusy = false
	m.summaryDone = nil
	token := m.fetchSeq
	return m, m.fetchCmd(token)
}

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	x, y := mouseXY(msg)
	action := mouseAction(msg)
	stackIndex, prIndex, hit := m.ballHit(x, y)
	switch action {
	case mousePress:
		if hit {
			m.choose(app.Selection{StackIndex: stackIndex, PRIndex: prIndex}, true)
			return m, m.startSelectedSummary()
		}
	case mouseMotion:
		if hit {
			m.choose(app.Selection{StackIndex: stackIndex, PRIndex: prIndex}, true)
			return m, m.startSelectedSummary()
		}
		m.State.CardVisible = false
	}
	return m, nil
}
