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
	titleX := paintHanami(c, 1, 0)
	c.text(titleX, 0, title, text, surface, max(1, inner-displayWidth(badge)-1-(titleX-1)))
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

	m.paintList(c, listWidth, mainTop, listBottom, surface, text, muted, stick)
	m.paintInspector(c, inspX, inspY, inspW, inspH, insp.Compact, raised, text, muted, surface)

	if m.State.Searching {
		c.fill(1, statusY, inner, 1, raised)
		query := m.State.Query
		placeholder := "filter stacks, PR titles, branches, numbers, author"
		if query == "" {
			c.text(1, statusY, placeholder, muted, raised, inner)
		} else {
			c.text(1, statusY, query, text, raised, inner)
		}
		m.paintSearchFooter(c, 1, footerY, inner, text, muted, surface)
	} else {
		status := m.sourceState()
		statusFg := muted
		if m.State.Feedback != "" {
			status = m.State.Feedback
			statusFg = text
		}
		c.text(1, statusY, status, statusFg, surface, inner)
		m.paintFooter(c, 1, footerY, inner, text, muted, surface)
	}

	return c.render()
}

type helpItem struct {
	key    string
	action string
	cta    bool
}

func (m Model) paintSearchFooter(c *canvas, x, y, maxWidth int, text, muted, surface string) {
	paintHelp(c, x, y, maxWidth, "  ", []helpItem{
		{key: "type", action: " to filter"},
		{key: "backspace", action: " edits"},
		{key: "esc", action: " clears / exits"},
	}, text, muted, surface)
}

func (m Model) paintFooter(c *canvas, x, y, maxWidth int, text, muted, surface string) {
	compact := IsCompact(m.Width)
	var items []helpItem
	switch {
	case m.Help && compact:
		items = []helpItem{
			{key: "enter", action: " go", cta: true},
			{key: "o", action: " open"},
			{key: "r", action: " sync"},
			{key: "esc"},
			{key: "q", action: " quit"},
		}
	case m.Help:
		items = []helpItem{
			{key: "enter", action: " checkout", cta: true},
			{key: "o", action: " open"},
			{key: "r", action: " refresh"},
			{key: "esc", action: " close"},
			{key: "q", action: " quit"},
		}
	case compact:
		items = []helpItem{
			{key: "↑↓", action: " stack"},
			{key: "←→", action: " layer"},
			{key: "/", action: " find"},
			{key: "?", action: " help"},
		}
	case m.Width <= 90:
		items = []helpItem{
			{key: "↑↓", action: " stack"},
			{key: "←→", action: " layer"},
			{key: "enter", action: " checkout", cta: true},
			{key: "/", action: " filter"},
			{key: "?", action: " help"},
			{key: "q", action: " quit"},
		}
	default:
		items = []helpItem{
			{key: "↑↓", action: " stack"},
			{key: "←→", action: " layer"},
			{key: "enter", action: " checkout", cta: true},
			{key: "o", action: " open"},
			{key: "r", action: " refresh"},
			{key: "/", action: " filter"},
			{key: "esc", action: " close"},
			{key: "?", action: " help"},
			{key: "q", action: " quit"},
		}
	}
	paintHelp(c, x, y, maxWidth, "  ", items, text, muted, surface)
}

func paintHelp(c *canvas, x, y, maxWidth int, sep string, items []helpItem, text, muted, surface string) {
	limit := x + maxWidth
	for i, item := range items {
		if x >= limit {
			return
		}
		if i > 0 {
			c.text(x, y, sep, muted, surface, limit-x)
			x += displayWidth(sep)
		}
		token := item.key + item.action
		if item.cta {
			c.textReverse(x, y, token, limit-x)
			x += displayWidth(clip(token, limit-x))
			continue
		}
		c.text(x, y, item.key, text, surface, limit-x)
		x += displayWidth(clip(item.key, limit-x))
		if item.action != "" && x < limit {
			c.text(x, y, item.action, muted, surface, limit-x)
			x += displayWidth(clip(item.action, limit-x))
		}
	}
}

func paintHanami(c *canvas, x, y int) int {
	for i, token := range []string{"ready", "queued", "merged"} {
		c.set(x+i, y, '●', domain.Color(token), "")
	}
	return x + 4
}

func (m Model) paintList(c *canvas, listWidth, top, bottom int, surface, text, muted, stick string) {
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
			glyph := '●'
			if selected {
				glyph = '◉'
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

func (m Model) paintInspector(c *canvas, x, y, w, h int, compact bool, raised, text, muted, surface string) {
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

	cardW := min(w, GetInspectorSize(TerminalSize{Width: m.Width, Height: m.Height}).Width)
	cardH := min(h, GetInspectorSize(TerminalSize{Width: m.Width, Height: m.Height}).Height)
	c.box(x, y, cardW, cardH, domain.Color("border"), raised, text)
	maxLine := max(8, cardW-4)
	state := domain.GetDisplayState(pr)
	headline := domain.DisplayStateLabel[state] + " · " + domain.DisplayStateDetail(pr)
	c.text(x+2, y+1, "#"+itoa(pr.Number)+" "+pr.Title, text, raised, maxLine)
	c.text(x+2, y+2, headline, muted, raised, maxLine)

	ci := ciLine(pr)
	review := reviewLine(pr)
	diff := fmt.Sprintf("+%d −%d · %d files", pr.Additions, pr.Deletions, pr.ChangedFiles)
	if compact {
		x3 := paintLabeled(c, x+2, y+3, maxLine, ci, text, muted, raised)
		if x3+2 < x+2+maxLine {
			c.text(x3, y+3, "  ", muted, raised, 2)
			paintLabeled(c, x3+2, y+3, max(0, x+2+maxLine-(x3+2)), review, text, muted, raised)
		}
		c.text(x+2, y+4, diff, text, raised, maxLine)
		c.text(x+2, y+5, pr.Branch, muted, raised, maxLine)
		paintHint(c, x+2, y+6, maxLine, text, muted, raised)
		return
	}
	paintLabeled(c, x+2, y+3, maxLine, ci, text, muted, raised)
	paintLabeled(c, x+2, y+4, maxLine, review, text, muted, raised)
	c.text(x+2, y+5, diff, text, raised, maxLine)
	c.text(x+2, y+6, pr.Branch, muted, raised, maxLine)
	paintHint(c, x+2, y+7, maxLine, text, muted, raised)
}

func paintHint(c *canvas, x, y, maxWidth int, text, muted, surface string) {
	paintHelp(c, x, y, maxWidth, "  ", []helpItem{
		{key: "click", action: " checkout"},
		{key: "o", action: " open"},
	}, text, muted, surface)
}

func paintLabeled(c *canvas, x, y, maxWidth int, line, valueFg, labelFg, bg string) int {
	if maxWidth <= 0 {
		return x
	}
	label, value, ok := cutWord(line)
	if !ok {
		c.text(x, y, line, labelFg, bg, maxWidth)
		return x + displayWidth(clip(line, maxWidth))
	}
	key := label + " "
	c.text(x, y, key, labelFg, bg, maxWidth)
	used := displayWidth(clip(key, maxWidth))
	if used >= maxWidth {
		return x + used
	}
	c.text(x+used, y, value, valueFg, bg, maxWidth-used)
	return x + used + displayWidth(clip(value, maxWidth-used))
}

func cutWord(s string) (word, rest string, ok bool) {
	for i, r := range s {
		if r == ' ' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
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
