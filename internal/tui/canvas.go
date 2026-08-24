package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/gsimone/dango-tui/internal/domain"
)

type cell struct {
	r  rune
	fg string
	bg string
}

type canvas struct {
	width  int
	height int
	cells  []cell
}

func newCanvas(width, height int, bg string) *canvas {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	n := width * height
	cells := make([]cell, n)
	text := domain.Color("text")
	proto := cell{r: ' ', fg: text, bg: bg}
	for i := range cells {
		cells[i] = proto
	}
	return &canvas{width: width, height: height, cells: cells}
}

func (c *canvas) at(x, y int) *cell {
	return &c.cells[y*c.width+x]
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
	cell := c.at(x, y)
	if r != 0 {
		cell.r = r
	}
	if fg != "" {
		cell.fg = fg
	}
	if bg != "" {
		cell.bg = bg
	}
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
	b.Grow(c.width * c.height * 24)
	for y := 0; y < c.height; y++ {
		if y > 0 {
			b.WriteByte('\n')
		}
		row := c.cells[y*c.width : (y+1)*c.width]
		x := 0
		for x < c.width {
			cur := row[x]
			j := x + 1
			for j < c.width && row[j].fg == cur.fg && row[j].bg == cur.bg {
				j++
			}
			writeTrueColor(&b, cur.fg, cur.bg)
			for i := x; i < j; i++ {
				b.WriteRune(row[i].r)
			}
			b.WriteString("\x1b[0m")
			x = j
		}
	}
	return b.String()
}

func writeTrueColor(b *strings.Builder, fg, bg string) {
	fr, fgv, fb, okFG := domain.ParseRGB(fg)
	br, bgv, bb, okBG := domain.ParseRGB(bg)
	b.WriteString("\x1b[")
	if okFG {
		b.WriteString("38;2;")
		writeUint(b, fr)
		b.WriteByte(';')
		writeUint(b, fgv)
		b.WriteByte(';')
		writeUint(b, fb)
	}
	if okBG {
		if okFG {
			b.WriteByte(';')
		}
		b.WriteString("48;2;")
		writeUint(b, br)
		b.WriteByte(';')
		writeUint(b, bgv)
		b.WriteByte(';')
		writeUint(b, bb)
	}
	b.WriteByte('m')
}

func writeUint(b *strings.Builder, n int) {
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	if n >= 100 {
		b.WriteByte(byte('0' + n/100))
		n %= 100
		b.WriteByte(byte('0' + n/10))
		b.WriteByte(byte('0' + n%10))
		return
	}
	if n >= 10 {
		b.WriteByte(byte('0' + n/10))
		b.WriteByte(byte('0' + n%10))
		return
	}
	b.WriteByte(byte('0' + n))
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
