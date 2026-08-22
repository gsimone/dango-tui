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
	r       rune
	fg      string
	bg      string
	reverse bool
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
	c.cells[y][x] = cell{r: r, fg: fg, bg: bg}
}

func (c *canvas) setReverse(x, y int, r rune) {
	if x < 0 || y < 0 || x >= c.width || y >= c.height {
		return
	}
	c.cells[y][x] = cell{r: r, reverse: true}
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

func (c *canvas) textReverse(x, y int, value string, maxWidth int) {
	if maxWidth <= 0 || y < 0 || y >= c.height {
		return
	}
	clipped := clip(value, maxWidth)
	i := 0
	for _, r := range clipped {
		if i >= maxWidth || x+i >= c.width {
			break
		}
		c.setReverse(x+i, y, r)
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
			for j < c.width && c.cells[y][j].fg == cur.fg && c.cells[y][j].bg == cur.bg && c.cells[y][j].reverse == cur.reverse {
				j++
			}
			var text strings.Builder
			for i := x; i < j; i++ {
				text.WriteRune(c.cells[y][i].r)
			}
			style := lipgloss.NewStyle()
			if cur.fg != "" {
				style = style.Foreground(lipgloss.Color(cur.fg))
			}
			if cur.bg != "" {
				style = style.Background(lipgloss.Color(cur.bg))
			}
			if cur.reverse {
				style = style.Reverse(true)
			}
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
