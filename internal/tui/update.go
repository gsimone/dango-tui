package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/app"
)

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
	case openResultMsg:
		if msg.err != nil {
			m.State.Feedback = "Could not open " + msg.url
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
	case "enter":
		if pr, ok := m.SelectedPR(); ok {
			m.checkout(pr)
		}
	case "o":
		if pr, ok := m.SelectedPR(); ok {
			return m, m.open(pr)
		}
	case "r":
		m.State.Feedback = "Fixture data refreshed · no network"
	case "/":
		m.State.Query = ""
		m.State.Searching = true
		m.State.CardVisible = false
	case "?":
		m.Help = !m.Help
	case "esc", "escape":
		m.State.CardVisible = false
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
			if stacks := m.Stacks(); stackIndex < len(stacks) && prIndex < len(stacks[stackIndex].PRs) {
				m.choose(app.Selection{StackIndex: stackIndex, PRIndex: prIndex}, true)
				m.checkout(stacks[stackIndex].PRs[prIndex])
			}
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
