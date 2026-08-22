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
	listWidth := ListPaneWidth(width)
	inner := innerWidth(width)

	// y=0 is the one blank row at the top.
	m.paintBrand(c, width, surface, meta, stick)
	metaLine := fmt.Sprintf("%d stacks / %d layers · local deterministic data", m.stackCount(), m.layerCount())
	c.text(PadX, PadTop+1, metaLine, meta, surface, inner)
	// two blank rows of air before the list (y=3,4).

	footerY := height - 1
	mainTop := ListStartY
	mainBottom := footerY
	if mainBottom < mainTop {
		mainBottom = mainTop
	}

	m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, paper, meta, stick)
	m.paintRule(c, mainTop, mainBottom, meta, surface)
	m.paintInspectorPane(c, insp, surface, paper, meta)

	if m.State.Searching {
		c.fill(PadX, footerY, inner, 1, raised)
		query := m.State.Query
		if query == "" {
			c.text(PadX, footerY, "type to filter  backspace edits  esc clears / exits", meta, raised, inner)
		} else {
			c.text(PadX, footerY, query, meta, raised, inner)
		}
	} else if m.State.Feedback != "" {
		c.text(PadX, footerY, m.State.Feedback, meta, surface, inner)
	} else {
		c.text(PadX, footerY, m.footer(), meta, surface, inner)
	}

	return c.render()
}

func (m Model) paintBrand(c *canvas, width int, surface, meta, stick string) {
	badge := "fixture"
	c.text(width-PadX-displayWidth(badge), PadTop, badge, meta, surface, displayWidth(badge))
	hues := []string{"ready", "open", "queued"}
	x := PadX
	for i, token := range hues {
		c.set(x, PadTop, '●', domain.Color(token), surface)
		if i < len(hues)-1 {
			c.set(x+1, PadTop, '-', stick, surface)
		}
		x += 2
	}
	c.text(PadX+6, PadTop, "DANGO", meta, surface, 5)
}

func (m Model) paintRule(c *canvas, top, bottom int, meta, surface string) {
	x := RuleX(m.Width)
	for y := top; y < bottom; y++ {
		c.set(x, y, '│', meta, surface)
	}
}

func (m Model) footer() string {
	compact := IsCompact(m.Width)
	if m.Help {
		if compact {
			return "↑↓ stack  enter  o  r  esc  q"
		}
		return "↑↓ stack  enter checkout  o open  r refresh  esc  q"
	}
	if compact {
		return "↑↓ stack  ←→ layer  /  ?  q"
	}
	if m.Width <= 90 {
		return "↑↓ stack  ←→ layer  enter checkout  o open  /  ?  q"
	}
	return "↑↓ stack  ←→ layer  enter checkout  o open  a add  r refresh  / filter  esc  ?  q"
}

func (m Model) paintList(c *canvas, listWidth, top, bottom int, surface, raised, paper, meta, stick string) {
	stacks := m.Stacks()
	if len(stacks) == 0 {
		c.text(PadX, top, m.emptyMessage(), meta, surface, max(1, listWidth))
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
			c.fill(PadX, y, listWidth, 1, rowBg)
		}
		marker := "· "
		if selectedStack {
			marker = "▸ "
		}
		c.text(PadX, y, clip(marker+stack.Name, layout.NameWidth), nameFg, rowBg, layout.NameWidth)

		ballX := PadX + layout.NameWidth + 1
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
			remain := max(0, PadX+listWidth-descX)
			c.text(descX, y, clip(stackHealth(stack)+" · "+stack.Description, remain), meta, rowBg, remain)
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
	row := 1
	if h <= row {
		return
	}
	if stack, ok := m.SelectedStack(); ok && stack.Summary != "" {
		c.text(x, y+row, stack.Summary, meta, surface, maxLine)
		row++
	}
	if h <= row {
		return
	}
	c.text(x, y+row, headline, meta, surface, maxLine)
	row++

	ci := ciLine(pr)
	review := reviewLine(pr)
	diff := fmt.Sprintf("+%d −%d · %d files", pr.Additions, pr.Deletions, pr.ChangedFiles)
	hint := "o open"
	if place.Compact {
		if h > row {
			c.text(x, y+row, ci+" · "+review, meta, surface, maxLine)
			row++
		}
		if h > row {
			c.text(x, y+row, diff, meta, surface, maxLine)
			row++
		}
		if h > row {
			c.text(x, y+row, pr.Branch, meta, surface, maxLine)
			row++
		}
		if h > row {
			c.text(x, y+row, hint, meta, surface, maxLine)
		}
		return
	}
	for _, line := range []string{ci, review, diff, pr.Branch, hint} {
		if row >= h {
			break
		}
		c.text(x, y+row, line, meta, surface, maxLine)
		row++
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
	listWidth := ListPaneWidth(m.Width)
	if y < ListStartY || y >= m.Height-1 || x < PadX || x >= PadX+listWidth {
		return 0, 0, false
	}
	stackIndex = y - ListStartY
	if stackIndex < 0 || stackIndex >= len(stacks) {
		return 0, 0, false
	}
	stack := stacks[stackIndex]
	layout := GetListRowLayout(listWidth, m.Width, len(stack.PRs))
	ballX := PadX + layout.NameWidth + 1
	if x < ballX || x >= ballX+layout.BallsWidth {
		return 0, 0, false
	}
	prIndex = (x - ballX) / 2
	if prIndex < 0 || prIndex >= len(stack.PRs) {
		return 0, 0, false
	}
	return stackIndex, prIndex, true
}
