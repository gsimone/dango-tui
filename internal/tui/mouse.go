package tui

import tea "github.com/charmbracelet/bubbletea"

type mouseKind int

const (
	mouseOther mouseKind = iota
	mouseMotion
	mousePress
)

func mouseXY(msg tea.MouseMsg) (int, int) {
	return msg.X, msg.Y
}

func mouseAction(msg tea.MouseMsg) mouseKind {
	// Bubble Tea v1 exposes both the newer Action field and the older Type alias.
	switch msg.Action {
	case tea.MouseActionMotion:
		return mouseMotion
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			return mousePress
		}
	}
	switch msg.Type {
	case tea.MouseMotion:
		return mouseMotion
	case tea.MouseLeft:
		return mousePress
	}
	return mouseOther
}
