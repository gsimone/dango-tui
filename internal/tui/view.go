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

	if StackedInspector(width) {
		m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, paper, meta, stick)
	} else {
		m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, paper, meta, stick)
		m.paintRule(c, mainTop, mainBottom, meta, surface)
		m.paintInspectorPane(c, insp, surface, paper, meta)
	}

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
	nameW := 8
	for _, stack := range stacks {
		n := displayWidth("· " + stack.Name)
		if n > nameW {
			nameW = n
		}
	}
	statusW := 8
	ballColW := 13
	nameW = min(nameW, max(8, listWidth-statusW-ballColW-2))
	statusX := PadX + nameW + 1 + ballColW + 1
	start := m.listOrigin(len(stacks), sel.StackIndex, top, bottom)
	y := top
	for i := start; i < len(stacks); i++ {
		stack := stacks[i]
		if y >= bottom {
			break
		}
		rowBg := surface
		nameFg := meta
		selectedStack := i == sel.StackIndex
		focus := 0
		if selectedStack {
			rowBg = raised
			nameFg = paper
			focus = sel.PRIndex
			c.fill(PadX, y, listWidth, 1, rowBg)
		}
		from, show, lead, tail := ballWindow(len(stack.PRs), focus)
		marker := "· "
		if selectedStack {
			marker = "▸ "
		}
		c.text(PadX, y, marker+stack.Name, nameFg, rowBg, nameW)

		ballX := PadX + nameW + 1
		x := ballX
		if lead {
			c.text(x, y, "...", meta, rowBg, 3)
			x += 3
			c.set(x, y, '-', stick, rowBg)
			x++
		}
		for i := 0; i < show; i++ {
			prIndex := from + i
			pr := stack.PRs[prIndex]
			state := domain.GetDisplayState(pr)
			fg := domain.Color(domain.StateColorToken(state))
			selected := selectedStack && prIndex == sel.PRIndex
			glyph := '○'
			if selected {
				glyph = '●'
			}
			c.set(x, y, glyph, fg, rowBg)
			x++
			if i < show-1 || tail {
				c.set(x, y, '-', stick, rowBg)
				x++
			}
		}
		if tail {
			c.text(x, y, "...", meta, rowBg, 3)
		}
		remain := max(0, PadX+listWidth-statusX)
		if remain >= 4 {
			c.text(statusX, y, clip(stackHealth(stack), remain), meta, rowBg, remain)
		}
		y++
		if StackedInspector(m.Width) && selectedStack && m.State.CardVisible {
			inspH := m.stackedPaneHeight(listWidth)
			if y+inspH > bottom {
				inspH = max(1, bottom-y)
			}
			place := CardPlacement{Left: PadX, Top: y, Width: max(1, listWidth), Height: inspH, Compact: IsCompact(m.Width)}
			m.paintInspectorPane(c, place, surface, paper, meta)
			y += inspH
		}
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

	maxLine := max(1, w)
	state := domain.GetDisplayState(pr)
	headline := domain.DisplayStateLabel[state] + " · " + domain.DisplayStateDetail(pr)
	title := "#" + itoa(pr.Number) + " " + pr.Title
	row := 0
	for _, line := range wrapWords(title, maxLine) {
		if row >= h {
			break
		}
		c.text(x, y+row, line, paper, surface, maxLine)
		row++
	}
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

func stackHealth(stack domain.Stack) string {
	if len(stack.PRs) == 0 {
		return "no layers"
	}
	head := stack.PRs[len(stack.PRs)-1]
	switch domain.GetDisplayState(head) {
	case domain.StateReady:
		return "ready"
	case domain.StateCIFailure:
		return "ci failed"
	case domain.StateReviewBlocked:
		return "blocked"
	case domain.StateQueued:
		return "queued"
	case domain.StateDraft:
		return "draft"
	case domain.StateMerged:
		return "merged"
	default:
		return "pending"
	}
}

func (m Model) listOrigin(n, selected, top, bottom int) int {
	room := max(1, bottom-top)
	if StackedInspector(m.Width) && m.State.CardVisible {
		room = max(1, room-8)
	}
	if selected < room {
		return 0
	}
	if selected > n-1 {
		selected = n - 1
	}
	return selected - room + 1
}

func (m Model) stackedPaneHeight(listWidth int) int {
	pr, ok := m.SelectedPR()
	if !ok {
		return 3
	}
	title := "#" + itoa(pr.Number) + " " + pr.Title
	lines := len(wrapWords(title, max(1, listWidth)))
	extra := 6
	if IsCompact(m.Width) {
		extra = 6
	}
	return max(3, lines+extra)
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
	sel := app.ClampSelection(m.State.Selection, stacks)
	start := m.listOrigin(len(stacks), sel.StackIndex, ListStartY, m.Height-1)
	rowY := ListStartY
	stackIndex = -1
	for i := start; i < len(stacks); i++ {
		if y == rowY {
			stackIndex = i
			break
		}
		rowY++
		if StackedInspector(m.Width) && i == sel.StackIndex && m.State.CardVisible {
			rowY += m.stackedPaneHeight(listWidth)
		}
	}
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

func ballWindow(n, focus int) (from, show int, lead, tail bool) {
	if n <= 5 {
		return 0, n, false, false
	}
	if focus < 0 {
		focus = 0
	}
	if focus > n-1 {
		focus = n - 1
	}
	from = (focus / 3) * 3
	if from > n-3 {
		from = n - 3
	}
	show = 3
	if from+show > n {
		show = n - from
	}
	return from, show, from > 0, from+show < n
}
