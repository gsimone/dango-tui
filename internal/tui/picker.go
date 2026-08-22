package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/summary"
)

func pickerRows(current summary.Provider) []summary.Provider {
	rows := summary.Catalog()
	if current.Empty() {
		return rows
	}
	for _, row := range rows {
		if summary.SameProvider(current, row) {
			return rows
		}
	}
	return append(append([]summary.Provider{}, rows...), current)
}

func pickIndex(current summary.Provider, rows []summary.Provider) int {
	for i, row := range rows {
		if summary.SameProvider(current, row) {
			return i
		}
	}
	return 0
}

func (m *Model) openPicker() {
	m.Help = false
	m.Picking = true
	m.PickIndex = pickIndex(m.Provider, pickerRows(m.Provider))
}

func (m *Model) cancelPicker() {
	m.Picking = false
}

func (m Model) selectPicker() (tea.Model, tea.Cmd) {
	rows := pickerRows(m.Provider)
	if m.PickIndex < 0 || m.PickIndex >= len(rows) {
		m.Picking = false
		return m, nil
	}
	next := rows[m.PickIndex]
	m.Picking = false
	if summary.SameProvider(m.Provider, next) {
		return m, nil
	}
	m.Provider = next
	m.fetchSeq++
	return m, m.startSummaries()
}

func (m Model) movePicker(delta int) Model {
	rows := pickerRows(m.Provider)
	if len(rows) == 0 {
		m.PickIndex = 0
		return m
	}
	m.PickIndex += delta
	if m.PickIndex < 0 {
		m.PickIndex = 0
	}
	if m.PickIndex > len(rows)-1 {
		m.PickIndex = len(rows) - 1
	}
	return m
}

func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	rows := pickerRows(m.Provider)
	switch msg.String() {
	case "up":
		return m.movePicker(-1), nil
	case "down":
		return m.movePicker(1), nil
	case "home":
		m.PickIndex = 0
	case "end":
		if n := len(rows); n > 0 {
			m.PickIndex = n - 1
		}
	case "enter":
		return m.selectPicker()
	case "esc", "escape":
		m.cancelPicker()
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "ctrl+c":
		return m, nil
	}
	return m, nil
}

func (m Model) paintPicker(c *canvas, top, bottom int, surface, raised, paper, meta string) {
	rows := pickerRows(m.Provider)
	index := m.PickIndex
	if index < 0 {
		index = 0
	}
	if n := len(rows); n > 0 && index >= n {
		index = n - 1
	}
	width := innerWidth(m.Width)
	y := top
	for i, row := range rows {
		if y >= bottom {
			break
		}
		selected := i == index
		rowBg := surface
		fg := meta
		marker := "· "
		if selected {
			rowBg = raised
			fg = paper
			marker = "▸ "
			c.fill(PadX, y, width, 1, rowBg)
		}
		c.text(PadX, y, marker+row.Label(), fg, rowBg, width)
		y++
	}
}

func (m Model) pickerFooter() string {
	return "[ ↑↓ ]  [ enter ]  [ esc ]"
}
