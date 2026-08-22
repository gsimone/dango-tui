package tui

import (
	"fmt"

	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/domain"
)

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	return m.renderFrame(m.Width, m.Height)
}

func (m Model) RenderFrame(width, height int) string {
	m.Width = width
	m.Height = height
	return m.renderFrame(width, height)
}

func (m Model) renderFrame(width, height int) string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	surface := domain.Color("surface")
	raised := domain.Color("surfaceRaised")
	paper := domain.Color("paper")
	meta := domain.Color("meta")
	stick := domain.Color("stick")

	c := newCanvas(width, height, surface)
	insp := GetInspectorSize(TerminalSize{Width: width, Height: height})
	listWidth := ListTerminalWidth(width, insp.Width)
	inner := max(1, width-2)

	m.paintBrand(c, width, surface, meta, stick)
	metaLine := fmt.Sprintf("%d stacks / %d layers · local deterministic data", m.stackCount(), m.layerCount())
	c.text(1, 1, metaLine, meta, surface, inner)
	// y=2 and y=3 stay empty: two blank rows of air.

	statusY := height - 2
	footerY := height - 1
	mainTop := ListStartY
	mainBottom := statusY
	if mainBottom < mainTop {
		mainBottom = mainTop
	}

	m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, paper, meta, stick)
	m.paintInspectorPane(c, insp, surface, paper, meta)

	if m.State.Searching {
		c.fill(1, statusY, inner, 1, raised)
		query := m.State.Query
		placeholder := "filter stacks, PR titles, branches, numbers, author"
		if query == "" {
			c.text(1, statusY, placeholder, meta, raised, inner)
		} else {
			c.text(1, statusY, query, paper, raised, inner)
		}
		c.text(1, footerY, "type to filter  backspace edits  esc clears / exits", meta, surface, inner)
	} else {
		status := m.sourceState()
		statusFg := meta
		if m.State.Feedback != "" {
			status = m.State.Feedback
			statusFg = paper
		}
		c.text(1, statusY, status, statusFg, surface, inner)
		c.text(1, footerY, m.footer(), meta, surface, inner)
	}

	return c.render()
}

func (m Model) paintBrand(c *canvas, width int, surface, meta, stick string) {
	badge := "fixture"
	c.text(width-1-displayWidth(badge), 0, badge, meta, surface, displayWidth(badge))
	// Packed balls are the mark. Color only on the balls; no wordmark.
	hues := []string{"ready", "open", "queued"}
	for i, token := range hues {
		x := 1 + i*2
		c.set(x, 0, '○', domain.Color(token), surface)
		if i < len(hues)-1 {
			c.set(x+1, 0, '-', stick, surface)
		}
	}
}

func (m Model) footer() string {
	compact := IsCompact(m.Width)
	if m.Help {
		if compact {
			return "enter go · o open · r sync · esc · q quit"
		}
		return "enter checkout · o open · r refresh · esc close · q quit"
	}
	full := "↑↓ stack  ←→ layer  enter checkout  o open  a add  r refresh  / filter  esc close  ? help  q quit"
	if compact {
		return "↑↓ stack · ←→ layer · / find · ? help"
	}
	if m.Width <= 90 {
		return "↑↓ stack · ←→ layer · enter checkout · / filter · a add · ? help · q quit"
	}
	return full
}

func (m Model) paintList(c *canvas, listWidth, top, bottom int, surface, raised, paper, meta, stick string) {
	stacks := m.Stacks()
	if len(stacks) == 0 {
		c.text(1, top, m.emptyMessage(), meta, surface, max(1, listWidth-2))
		return
	}
	sel := app.ClampSelection(m.State.Selection, stacks)
	maxRows := max(0, bottom-top)
	for i, stack := range stacks {
		if i >= maxRows {
			break
		}
		y := top + i
		layout := GetListRowLayout(listWidth, m.Width, len(stack.PRs))
		rowBg := surface
		nameFg := meta
		selectedStack := i == sel.StackIndex
		if selectedStack {
			rowBg = raised
			nameFg = paper
			c.fill(1, y, max(1, listWidth-2), 1, rowBg)
		}
		marker := "· "
		if selectedStack {
			marker = "▸ "
		}
		c.text(1, y, clip(marker+stack.Name, layout.NameWidth), nameFg, rowBg, layout.NameWidth)

		ballX := 1 + layout.NameWidth + 1
		for prIndex, pr := range stack.PRs {
			state := domain.GetDisplayState(pr)
			fg := domain.Color(domain.StateColorToken(state))
			selected := selectedStack && prIndex == sel.PRIndex
			glyph := '○'
			if selected {
				glyph = '●'
			}
			c.set(ballX+prIndex*2, y, glyph, fg, rowBg)
			connector := '-'
			if prIndex == len(stack.PRs)-1 {
				connector = ' '
			}
			c.set(ballX+prIndex*2+1, y, connector, stick, rowBg)
		}

		if !layout.Compact {
			descX := ballX + layout.BallsWidth + 1
			remain := max(0, listWidth-1-descX)
			descFg := meta
			if selectedStack {
				descFg = paper
			}
			c.text(descX, y, clip(stackHealth(stack)+" · "+stack.Description, remain), descFg, rowBg, remain)
		}
	}
}

func stackHealth(stack domain.Stack) string {
	if len(stack.PRs) == 0 {
		return "no layers"
	}
	head := stack.PRs[len(stack.PRs)-1]
	switch domain.GetDisplayState(head) {
	case domain.StateReady:
		return "head ready"
	case domain.StateCIFailure:
		return "head CI failed"
	case domain.StateReviewBlocked:
		return "head blocked"
	case domain.StateQueued:
		return "head queued"
	case domain.StateDraft:
		return "head draft"
	case domain.StateMerged:
		return "merged"
	default:
		return "head pending"
	}
}

func (m Model) paintInspectorPane(c *canvas, place CardPlacement, surface, paper, meta string) {
	x, y, w, h := place.Left, place.Top, place.Width, place.Height
	if w < 1 || h < 1 {
		return
	}
	c.fill(x, y, w, h, surface)
	pr, ok := m.SelectedPR()
	if !m.State.CardVisible || !ok {
		return
	}

	maxLine := max(8, w)
	state := domain.GetDisplayState(pr)
	headline := domain.DisplayStateLabel[state] + " · " + domain.DisplayStateDetail(pr)
	c.text(x, y, "#"+itoa(pr.Number)+" "+pr.Title, paper, surface, maxLine)
	if h < 2 {
		return
	}
	c.text(x, y+1, headline, meta, surface, maxLine)

	ci := ciLine(pr)
	review := reviewLine(pr)
	diff := fmt.Sprintf("+%d −%d · %d files", pr.Additions, pr.Deletions, pr.ChangedFiles)
	hint := "o open"
	if place.Compact {
		if h > 2 {
			c.text(x, y+2, ci+" · "+review, meta, surface, maxLine)
		}
		if h > 3 {
			c.text(x, y+3, diff, meta, surface, maxLine)
		}
		if h > 4 {
			c.text(x, y+4, pr.Branch, meta, surface, maxLine)
		}
		if h > 5 {
			c.text(x, y+5, hint, meta, surface, maxLine)
		}
		return
	}
	lines := []string{ci, review, diff, pr.Branch, hint}
	for i, line := range lines {
		row := y + 2 + i
		if row >= y+h {
			break
		}
		c.text(x, row, line, meta, surface, maxLine)
	}
}

func ciLine(pr domain.PullRequest) string {
	switch pr.CI.State {
	case domain.CISuccess:
		total := itoa(pr.CI.Total)
		if pr.CI.Total == 0 {
			total = "all"
		}
		return "CI ✓ " + total + " checks"
	case domain.CIFailure:
		failed := pr.CI.Failed
		if failed == 0 {
			failed = 1
		}
		return fmt.Sprintf("CI × %d failed · %d total", failed, pr.CI.Total)
	case domain.CIPending:
		pending := pr.CI.Pending
		if pending == 0 {
			pending = 1
		}
		return fmt.Sprintf("CI ◌ %d pending · %d total", pending, pr.CI.Total)
	default:
		return "CI — not reported"
	}
}

func reviewLine(pr domain.PullRequest) string {
	if pr.Mergeable != nil && !*pr.Mergeable {
		return "Review ! merge conflict"
	}
	if pr.ChangesRequested {
		return "Review ! changes requested"
	}
	if pr.Approvals > 0 {
		noun := "approvals"
		if pr.Approvals == 1 {
			noun = "approval"
		}
		return fmt.Sprintf("Review ✓ %d %s", pr.Approvals, noun)
	}
	return "Review ◌ no decision yet"
}

func (m Model) ballHit(x, y int) (stackIndex, prIndex int, ok bool) {
	stacks := m.Stacks()
	if len(stacks) == 0 {
		return 0, 0, false
	}
	listWidth := ListTerminalWidth(m.Width, InspectorColumnWidth(m.Width))
	if y < ListStartY || y >= m.Height-2 || x >= listWidth {
		return 0, 0, false
	}
	stackIndex = y - ListStartY
	if stackIndex < 0 || stackIndex >= len(stacks) {
		return 0, 0, false
	}
	stack := stacks[stackIndex]
	layout := GetListRowLayout(listWidth, m.Width, len(stack.PRs))
	ballX := RootPaddingX + layout.NameWidth + 1
	if x < ballX || x >= ballX+layout.BallsWidth {
		return 0, 0, false
	}
	prIndex = (x - ballX) / 2
	if prIndex < 0 || prIndex >= len(stack.PRs) {
		return 0, 0, false
	}
	return stackIndex, prIndex, true
}
