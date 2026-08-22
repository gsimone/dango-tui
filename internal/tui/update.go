package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/app"
)

type fetchDoneMsg struct{}

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
		m.Fetching = false
		m.Fetched = "last fetched 2 mins ago"
		m.State.Feedback = ""
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
		if pr, ok := m.SelectedPR(); ok {
			return m, m.copyBranch(pr)
		}
	case "a":
		m.State.Feedback = "add · not wired"
	case "r":
		m.Fetching = true
		m.State.Feedback = ""
		return m, tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg { return fetchDoneMsg{} })
	case "/":
		m.Help = false
		m.State.Query = ""
		m.State.Searching = true
		m.State.CardVisible = false
	case "?":
		m.Help = !m.Help
	case "esc", "escape":
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
