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
	mainBottom := footerY - 1
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

	footX := PadX + 2
	footW := max(1, inner-4)
	if m.State.Searching {
		query := m.State.Query
		if query == "" {
			c.text(footX, footerY, "/", meta, surface, footW)
		} else {
			c.text(footX, footerY, "/"+query, meta, surface, footW)
		}
	} else if m.State.Feedback != "" {
		c.text(footX, footerY, m.State.Feedback, meta, surface, footW)
	} else {
		c.text(footX, footerY, m.footer(), meta, surface, footW)
	}

	return c.render()
}

func (m Model) paintBrand(c *canvas, width int, surface, meta, stick string) {
	badge := m.fetchBadge()
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
	repo := "example/stacks"
	c.text(PadX+6, PadTop, repo, meta, surface, displayWidth(repo))
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
	layout := GetListRowLayout(listWidth, m.Width, 0)
	nameW := layout.NameWidth
	statusW := layout.StatusWidth
	statusX := PadX + nameW + 1 + layout.BallsWidth + 1
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
		cells := ballCells(len(stack.PRs), focus)
		marker := "· "
		if selectedStack {
			marker = "▸ "
		}
		c.text(PadX, y, marker+stack.Name, nameFg, rowBg, nameW)

		ballX := PadX + nameW + 1
		x := ballX
		n := len(stack.PRs)
		for i, cell := range cells {
			if cell.pager {
				fg := meta
				if selectedStack && focus >= 2 && focus <= n-3 {
					fg = paper
				}
				c.text(x, y, "‹›", fg, rowBg, 2)
				x += 2
			} else if n <= 5 {
				pr := stack.PRs[cell.pr]
				state := domain.GetDisplayState(pr)
				fg := domain.Color(domain.StateColorToken(state))
				selected := selectedStack && cell.pr == sel.PRIndex
				glyph := '○'
				if selected {
					glyph = '●'
				}
				c.set(x, y, glyph, fg, rowBg)
				x++
			} else {
				pr := stack.PRs[cell.pr]
				state := domain.GetDisplayState(pr)
				fg := domain.Color(domain.StateColorToken(state))
				if selectedStack && cell.pr == sel.PRIndex {
					fg = paper
				}
				label := "(" + itoa(cell.pr+1) + ")"
				c.text(x, y, label, fg, rowBg, displayWidth(label))
				x += displayWidth(label)
			}
			if i < len(cells)-1 {
				c.set(x, y, '-', stick, rowBg)
				x++
			}
		}
		remain := min(statusW, max(0, PadX+listWidth-statusX))
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
	cx := ballX
	for _, cell := range ballCells(len(stack.PRs), sel.PRIndex) {
		w := cell.width(len(stack.PRs))
		if x >= cx && x < cx+w+1 {
			if cell.pager {
				prIndex = sel.PRIndex
				if prIndex < 2 || prIndex > len(stack.PRs)-3 {
					prIndex = 2
				}
			} else {
				prIndex = cell.pr
			}
			return stackIndex, prIndex, true
		}
		cx += w + 1
	}
	return 0, 0, false
}

type ballCell struct {
	pr    int
	pager bool
}

func (cell ballCell) width(n int) int {
	if cell.pager {
		return 2
	}
	if n <= 5 {
		return 1
	}
	return displayWidth("(" + itoa(cell.pr+1) + ")")
}

func ballCells(n, focus int) []ballCell {
	if n <= 5 {
		out := make([]ballCell, n)
		for i := 0; i < n; i++ {
			out[i].pr = i
		}
		return out
	}
	return []ballCell{{pr: 0}, {pr: 1}, {pager: true}, {pr: n - 2}, {pr: n - 1}}
}

func ballCellX(ballX, n, prIndex int) int {
	x := ballX
	for _, cell := range ballCells(n, prIndex) {
		if cell.pager && prIndex >= 2 && prIndex <= n-3 {
			return x
		}
		if !cell.pager && cell.pr == prIndex {
			return x
		}
		x += cell.width(n) + 1
	}
	return ballX
}
