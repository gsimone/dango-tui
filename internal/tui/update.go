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

type summaryDoneMsg struct {
	token       int
	id          string
	title       string
	description string
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
		return m.handleMouse(msg), nil
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
				return m, m.startSummaries()
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
	case summaryDoneMsg:
		m.applySummary(msg)
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
	case "down":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirDown), true)
	case "left":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirLeft), true)
	case "right":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirRight), true)
	case "home":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirHome), true)
	case "end":
		m.choose(app.MoveSelection(m.State.Selection, m.Stacks(), app.DirEnd), true)
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

func (m *Model) applySummary(msg summaryDoneMsg) {
	if msg.token != m.fetchSeq {
		return
	}
	title := strings.TrimSpace(msg.title)
	desc := strings.TrimSpace(msg.description)
	if title == "" && desc == "" {
		return
	}
	for i := range m.stacks {
		if m.stacks[i].ID != msg.id {
			continue
		}
		if title != "" {
			m.stacks[i].Name = title
			m.stacks[i].Summary = title
		}
		if desc != "" {
			m.stacks[i].Description = desc
		}
		return
	}
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
	token := m.fetchSeq
	return m, m.fetchCmd(token)
}

func (m Model) handleMouse(msg tea.MouseMsg) Model {
	x, y := mouseXY(msg)
	action := mouseAction(msg)
	stackIndex, prIndex, hit := m.ballHit(x, y)
	switch action {
	case mousePress:
		if hit {
			m.choose(app.Selection{StackIndex: stackIndex, PRIndex: prIndex}, true)
		}
	case mouseMotion:
		if hit {
			m.choose(app.Selection{StackIndex: stackIndex, PRIndex: prIndex}, true)
		} else {
			m.State.CardVisible = false
		}
	}
	return m
}
