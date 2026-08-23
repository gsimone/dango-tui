package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/muesli/termenv"
)

func init() {
	// Keep the OKLCH-derived palette even when stdout is not a TTY (tests, --frame).
	lipgloss.SetColorProfile(termenv.TrueColor)
}

type cell struct {
	r  rune
	fg string
	bg string
}

type canvas struct {
	width  int
	height int
	cells  [][]cell
}

func newCanvas(width, height int, bg string) *canvas {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	cells := make([][]cell, height)
	for y := 0; y < height; y++ {
		row := make([]cell, width)
		for x := 0; x < width; x++ {
			row[x] = cell{r: ' ', fg: domain.Color("text"), bg: bg}
		}
		cells[y] = row
	}
	return &canvas{width: width, height: height, cells: cells}
}

func (c *canvas) fill(x, y, w, h int, bg string) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			c.set(x+dx, y+dy, ' ', "", bg)
		}
	}
}

func (c *canvas) set(x, y int, r rune, fg, bg string) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	cell := c.cells[y][x]
	if r != 0 {
		cell.r = r
	}
	if fg != "" {
		cell.fg = fg
	}
	if bg != "" {
		cell.bg = bg
	}
	c.cells[y][x] = cell
}

func (c *canvas) text(x, y int, value string, fg, bg string, maxWidth int) {
	if maxWidth <= 0 || y < 0 || y >= c.height {
		return
	}
	clipped := clip(value, maxWidth)
	i := 0
	for _, r := range clipped {
		if i >= maxWidth || x+i >= c.width {
			break
		}
		c.set(x+i, y, r, fg, bg)
		i++
	}
}

func (c *canvas) box(x, y, w, h int, border, fill, titleFg string) {
	if w < 2 || h < 2 {
		return
	}
	c.fill(x, y, w, h, fill)
	for dx := 1; dx < w-1; dx++ {
		c.set(x+dx, y, '─', border, fill)
		c.set(x+dx, y+h-1, '─', border, fill)
	}
	for dy := 1; dy < h-1; dy++ {
		c.set(x, y+dy, '│', border, fill)
		c.set(x+w-1, y+dy, '│', border, fill)
	}
	c.set(x, y, '┌', border, fill)
	c.set(x+w-1, y, '┐', border, fill)
	c.set(x, y+h-1, '└', border, fill)
	c.set(x+w-1, y+h-1, '┘', border, fill)
	_ = titleFg
}

func (c *canvas) render() string {
	var b strings.Builder
	for y := 0; y < c.height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		x := 0
		for x < c.width {
			cur := c.cells[y][x]
			j := x + 1
			for j < c.width && c.cells[y][j].fg == cur.fg && c.cells[y][j].bg == cur.bg {
				j++
			}
			var text strings.Builder
			for i := x; i < j; i++ {
				text.WriteRune(c.cells[y][i].r)
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(cur.fg)).Background(lipgloss.Color(cur.bg))
			b.WriteString(style.Render(text.String()))
			x = j
		}
	}
	return b.String()
}

func clip(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if utf8.RuneCountInString(value) <= maxWidth {
		return value
	}
	if maxWidth == 1 {
		return "…"
	}
	runes := []rune(value)
	return string(runes[:maxWidth-1]) + "…"
}

func displayWidth(value string) int {
	return utf8.RuneCountInString(value)
}

func wrapWords(value string, width int) []string {
	if width < 1 {
		return nil
	}
	var lines []string
	var cur string
	flush := func() {
		if cur != "" {
			lines = append(lines, cur)
			cur = ""
		}
	}
	for _, word := range strings.Fields(value) {
		for utf8.RuneCountInString(word) > width {
			flush()
			runes := []rune(word)
			lines = append(lines, string(runes[:width]))
			word = string(runes[width:])
		}
		if cur == "" {
			cur = word
			continue
		}
		if utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(word) <= width {
			cur += " " + word
			continue
		}
		flush()
		cur = word
	}
	flush()
	return lines
}
