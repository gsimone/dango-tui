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
	border := domain.Color("border")
	text := domain.Color("text")
	muted := domain.Color("muted")
	focus := domain.Color("focus")
	stick := domain.Color("stick")

	c := newCanvas(width, height, surface)
	compact := IsCompact(width)
	listWidth := width

	badge := "fixture"
	title := m.title()
	if !compact {
		title = title + " / example/stacks"
	}
	inner := max(1, width-2)
	c.text(1, 0, title, text, surface, inner-displayWidth(badge)-1)
	c.text(width-1-displayWidth(badge), 0, badge, muted, surface, displayWidth(badge))

	meta := fmt.Sprintf("%d stacks / %d layers · local deterministic data", m.stackCount(), m.layerCount())
	c.text(1, 1, meta, muted, surface, inner)

	col := "STACK · BASE → HEAD"
	if !compact {
		col = "STACK                         LAYERS · BASE → HEAD"
	}
	c.text(1, 2, col, border, surface, inner)

	statusY := height - 2
	footerY := height - 1
	mainTop := 3
	mainBottom := statusY
	if mainBottom < mainTop {
		mainBottom = mainTop
	}

	m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, text, muted, stick)
	m.paintPostcard(c)

	if m.State.Searching {
		c.fill(1, statusY, inner, 1, raised)
		query := m.State.Query
		placeholder := "filter stacks, PR titles, branches, numbers, author"
		if query == "" {
			c.text(1, statusY, placeholder, muted, raised, inner)
		} else {
			c.text(1, statusY, query, text, raised, inner)
		}
		c.text(1, footerY, "type to filter  backspace edits  esc clears / exits", muted, surface, inner)
	} else {
		status := m.sourceState()
		statusFg := muted
		if m.State.Feedback != "" {
			status = m.State.Feedback
			statusFg = focus
		}
		c.text(1, statusY, status, statusFg, surface, inner)
		c.text(1, footerY, m.footer(), muted, surface, inner)
	}

	return c.render()
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

func (m Model) paintList(c *canvas, listWidth, top, bottom int, surface, raised, text, muted, stick string) {
	stacks := m.Stacks()
	if len(stacks) == 0 {
		c.text(1, top, m.emptyMessage(), muted, surface, max(1, listWidth-2))
		return
	}
	sel := app.ClampSelection(m.State.Selection, stacks)
	maxRows := max(0, bottom-top)
	for i, stack := range stacks {
		if i >= maxRows {
			break
		}
		y := top + i
		layout := GetRowLayout(listWidth, len(stack.PRs))
		rowBg := surface
		nameFg := muted
		selectedStack := i == sel.StackIndex
		if selectedStack {
			rowBg = raised
			nameFg = text
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
			bg := rowBg
			if selected {
				glyph = '●'
			}
			c.set(ballX+prIndex*2, y, glyph, fg, bg)
			connector := '-'
			if prIndex == len(stack.PRs)-1 {
				connector = ' '
			}
			c.set(ballX+prIndex*2+1, y, connector, stick, bg)
		}

		if !layout.Compact {
			descX := ballX + layout.BallsWidth + 1
			remain := max(0, (listWidth-1)-descX)
			c.text(descX, y, clip(stackHealth(stack)+" · "+stack.Description, remain), muted, rowBg, remain)
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

func (m Model) postcardPlace() (CardPlacement, bool) {
	anchor, ok := m.selectedBallAnchor()
	if !ok {
		return CardPlacement{}, false
	}
	size := TerminalSize{Width: m.Width, Height: m.Height}
	afterBalls := 0
	for _, stack := range m.Stacks() {
		layout := GetRowLayout(size.Width, len(stack.PRs))
		edge := RootPaddingX + layout.NameWidth + 1 + layout.BallsWidth + 1
		if edge > afterBalls {
			afterBalls = edge
		}
	}
	compact := IsCompact(size.Width)
	cardWidth := max(16, min(func() int {
		if compact {
			return 38
		}
		return 56
	}(), size.Width-2))
	cardHeight := 9
	if compact {
		cardHeight = 8
	}
	usableBottom := max(1, size.Height-2)
	listBottom := ListStartY + len(m.Stacks())

	// Prefer a postcard on the field below the stacks so the ball chain stays intact.
	if listBottom+cardHeight <= usableBottom {
		return keepCardOnScreen(size, CardPlacement{Left: 1, Top: listBottom, Width: cardWidth, Height: cardHeight, Compact: compact}), true
	}
	if afterBalls+cardWidth <= size.Width-1 {
		top := anchor.Y
		if top+cardHeight > usableBottom {
			top = max(1, usableBottom-cardHeight)
		}
		return keepCardOnScreen(size, CardPlacement{Left: afterBalls, Top: top, Width: cardWidth, Height: cardHeight, Compact: compact}), true
	}
	return ClampCardPlacement(size, anchor), true
}

func keepCardOnScreen(size TerminalSize, place CardPlacement) CardPlacement {
	usableBottom := max(1, size.Height-2)
	if place.Left < 1 {
		place.Left = 1
	}
	if place.Left+place.Width > size.Width-1 {
		place.Width = max(16, size.Width-1-place.Left)
	}
	if place.Top < 1 {
		place.Top = 1
	}
	if place.Top+place.Height > usableBottom {
		place.Height = max(3, usableBottom-place.Top)
	}
	return place
}

func (m Model) selectedBallAnchor() (struct{ X, Y int }, bool) {
	stack, ok := m.SelectedStack()
	if !ok {
		return struct{ X, Y int }{}, false
	}
	sel := app.ClampSelection(m.State.Selection, m.Stacks())
	return GetBallPoint(TerminalSize{Width: m.Width, Height: m.Height}, sel.StackIndex, sel.PRIndex, len(stack.PRs)), true
}

func (m Model) paintPostcard(c *canvas) {
	pr, ok := m.SelectedPR()
	if !m.State.CardVisible || !ok {
		return
	}
	place, ok := m.postcardPlace()
	if !ok {
		return
	}
	x, y, w, h := place.Left, place.Top, place.Width, place.Height
	if w < 2 || h < 2 {
		return
	}

	paper := domain.Color("postcard")
	ink := domain.Color("postcardInk")
	quiet := domain.Color("postcardMuted")
	edge := domain.Color("postcardEdge")
	c.box(x, y, w, h, edge, paper, ink)

	maxLine := max(8, w-4)
	state := domain.GetDisplayState(pr)
	stateFg := domain.Color(domain.StateColorToken(state))
	headline := domain.DisplayStateLabel[state] + " · " + domain.DisplayStateDetail(pr)
	c.text(x+2, y+1, "#"+itoa(pr.Number)+" "+pr.Title, ink, paper, maxLine)
	c.text(x+2, y+2, headline, stateFg, paper, maxLine)

	ci := ciLine(pr)
	review := reviewLine(pr)
	diff := fmt.Sprintf("+%d −%d · %d files", pr.Additions, pr.Deletions, pr.ChangedFiles)
	hint := "o open"
	if place.Compact {
		c.text(x+2, y+3, ci+" · "+review, quiet, paper, maxLine)
		c.text(x+2, y+4, diff, ink, paper, maxLine)
		c.text(x+2, y+5, pr.Branch, quiet, paper, maxLine)
		c.text(x+2, y+6, hint, ink, paper, maxLine)
		return
	}
	c.text(x+2, y+3, ci, quiet, paper, maxLine)
	c.text(x+2, y+4, review, quiet, paper, maxLine)
	c.text(x+2, y+5, diff, ink, paper, maxLine)
	c.text(x+2, y+6, pr.Branch, quiet, paper, maxLine)
	c.text(x+2, y+7, hint, ink, paper, maxLine)
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
	listWidth := ListTerminalWidth(m.Width, 0)
	statusY := m.Height - 2
	mainTop := ListStartY
	if y < mainTop || y >= statusY {
		return 0, 0, false
	}
	stackIndex = y - mainTop
	if stackIndex < 0 || stackIndex >= len(stacks) {
		return 0, 0, false
	}
	stack := stacks[stackIndex]
	layout := GetRowLayout(listWidth, len(stack.PRs))
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
