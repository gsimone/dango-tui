package tui

import (
	"os/exec"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/domain"
)

type openResultMsg struct {
	url string
	err error
}

func (m *Model) open(pr domain.PullRequest) tea.Cmd {
	m.State.CardVisible = true
	if pr.URL == "" {
		m.State.Feedback = "No URL on #" + itoa(pr.Number)
		return m.clearFeedback()
	}
	m.State.Feedback = "Opening " + pr.URL
	return openURLCmd(pr.URL)
}

func openURLCmd(raw string) tea.Cmd {
	return func() tea.Msg {
		return openResultMsg{url: raw, err: startBrowser(raw)}
	}
}

func startBrowser(raw string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", raw)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", raw)
	default:
		cmd = exec.Command("xdg-open", raw)
	}
	return cmd.Start()
}
