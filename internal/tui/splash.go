package tui

import (
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// dangoBlock is a 3-row ░▒▓█ DANGO. Not a 5-row figlet.
var dangoBlock = buildDangoBlock()

func buildDangoBlock() [3]string {
	letters := [][3]string{
		{"███░", "█ ▓█", "███░"},
		{"░██░", "█▒▓█", "█  █"},
		{"█  █", "█▓▒█", "█ ▓█"},
		{"░███", "█   ", "░█▓█"},
		{"░██░", "█  █", "░██░"},
	}
	var out [3]string
	for row := 0; row < 3; row++ {
		var b strings.Builder
		for i, letter := range letters {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(letter[row])
		}
		out[row] = b.String()
	}
	return out
}

func (m Model) splash() bool {
	return m.Live && len(m.stacks) == 0 && (m.waiting() || m.fetchErr != nil)
}

func (m Model) splashStatusLines(maxW int) []string {
	if m.fetchErr != nil && !m.Fetching {
		return wrapWords(m.errorCopyText(), maxW)
	}
	if m.splashKeep != "" {
		return wrapWords(m.splashKeep, maxW)
	}
	return []string{"fetching " + m.repoLabel()}
}

func (m Model) paintSplash(c *canvas, width, height int, surface, paper, meta, stick string) {
	blockW := displayWidth(dangoBlock[0])
	maxW := max(1, width-4)
	status := m.splashStatusLines(maxW)
	sha := m.splashSHA()
	shaLines := 0
	if sha != "" {
		shaLines = 1
	}
	descLine := m.splashDescribeLine()
	contentH := 5 + len(status) + shaLines + 1
	startY := (height - contentH) / 2
	if startY < 0 {
		startY = 0
	}
	letterX := (width - blockW) / 2
	if letterX < 0 {
		letterX = 0
	}
	for i, row := range dangoBlock {
		c.text(letterX, startY+i, row, paper, surface, max(1, width-letterX))
	}
	dotsY := startY + 4
	dotsX := (width - 5) / 2
	if dotsX < 0 {
		dotsX = 0
	}
	x := dotsX
	for i, token := range m.LogoDots {
		if token == "" || !domain.IsLogoToken(token) {
			token = domain.LogoTokens[i]
		}
		c.set(x, dotsY, '●', domain.Color(token), surface)
		if i < 2 {
			c.set(x+1, dotsY, '-', stick, surface)
		}
		x += 2
	}
	fg := meta
	if m.fetchErr != nil && !m.Fetching {
		fg = paper
	}
	for i, line := range status {
		fetchY := startY + 5 + i
		fetchX := (width - displayWidth(line)) / 2
		if fetchX < 0 {
			fetchX = 0
		}
		c.text(fetchX, fetchY, line, fg, surface, max(1, width-fetchX))
	}
	if sha != "" {
		shaY := startY + 5 + len(status)
		shaX := (width - displayWidth(sha)) / 2
		if shaX < 0 {
			shaX = 0
		}
		c.text(shaX, shaY, sha, meta, surface, max(1, width-shaX))
	}
	descY := startY + 5 + len(status) + shaLines
	descX := (width - displayWidth(descLine)) / 2
	if descX < 0 {
		descX = 0
	}
	c.text(descX, descY, descLine, meta, surface, max(1, width-descX))
}

func (m Model) splashDescribeLine() string {
	if d := strings.TrimSpace(m.Describe); d != "" {
		return "describe: " + d
	}
	return "describe: none"
}
