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
	wide := IsWide(width)
	insp := GetInspectorSize(TerminalSize{Width: width, Height: height})
	listWidth := ListTerminalWidth(width, insp.Width)

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

	var listBottom, inspX, inspY, inspW, inspH int
	if wide {
		inspW = insp.Width
		inspH = mainBottom - mainTop
		inspX = width - 1 - inspW
		inspY = mainTop
		listBottom = mainBottom
	} else {
		inspW = insp.Width
		inspH = insp.Height
		inspX = 1
		inspY = mainBottom - inspH
		if inspY < mainTop {
			inspY = mainTop
			inspH = mainBottom - mainTop
		}
		listBottom = inspY
	}

	m.paintList(c, listWidth, mainTop, listBottom, surface, raised, text, muted, stick)
	m.paintInspector(c, inspX, inspY, inspW, inspH, insp.Compact, raised, focus, text, muted, surface)

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
			nameFg = text
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
			c.set(ballX+prIndex*2+1, y, connector, muted, bg)
		}

		if !layout.Compact {
			descX := ballX + layout.BallsWidth + 1
			remain := max(0, 1+max(1, listWidth-2)-descX)
			if IsWide(m.Width) {
				insp := GetInspectorSize(TerminalSize{Width: m.Width, Height: m.Height})
				limit := m.Width - 1 - insp.Width - 1
				if descX+remain > limit {
					remain = max(0, limit-descX)
				}
			} else {
				remain = max(0, (listWidth-1)-descX)
			}
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

func (m Model) paintInspector(c *canvas, x, y, w, h int, compact bool, raised, focus, text, muted, surface string) {
	if w < 1 || h < 1 {
		return
	}
	c.fill(x, y, w, h, surface)
	pr, ok := m.SelectedPR()
	if !m.State.CardVisible || !ok {
		msg := "Select or hover a layer to inspect."
		if compact {
			msg = "Select a layer to inspect."
		}
		c.text(x, y, msg, muted, surface, w)
		return
	}

	maxLine := max(8, w)
	state := domain.GetDisplayState(pr)
	headline := domain.DisplayStateLabel[state] + " · " + domain.DisplayStateDetail(pr)
	c.text(x, y, "#"+itoa(pr.Number)+" "+pr.Title, text, surface, maxLine)
	c.text(x, y+1, headline, muted, surface, maxLine)

	ci := ciLine(pr)
	review := reviewLine(pr)
	diff := fmt.Sprintf("+%d −%d · %d files", pr.Additions, pr.Deletions, pr.ChangedFiles)
	hint := "o open"
	if compact {
		c.text(x, y+2, ci+" · "+review, muted, surface, maxLine)
		c.text(x, y+3, diff, muted, surface, maxLine)
		c.text(x, y+4, pr.Branch, muted, surface, maxLine)
		c.text(x, y+5, hint, muted, surface, maxLine)
		return
	}
	c.text(x, y+2, ci, muted, surface, maxLine)
	c.text(x, y+3, review, muted, surface, maxLine)
	c.text(x, y+4, diff, muted, surface, maxLine)
	c.text(x, y+5, pr.Branch, muted, surface, maxLine)
	c.text(x, y+6, hint, muted, surface, maxLine)
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
	insp := GetInspectorSize(TerminalSize{Width: m.Width, Height: m.Height})
	listWidth := ListTerminalWidth(m.Width, insp.Width)
	statusY := m.Height - 2
	mainTop := ListStartY
	mainBottom := statusY
	listBottom := mainBottom
	if !IsWide(m.Width) {
		listBottom = mainBottom - insp.Height
		if listBottom < mainTop {
			listBottom = mainTop
		}
	}
	if y < mainTop || y >= listBottom {
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
