package tui

import (
	"fmt"
	"strings"

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
	// one blank row under the header, then the list.

	footerY := height - 1
	mainTop := ListStartY
	mainBottom := ListBottomY(height)
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
	if m.Help {
		m.paintHelp(c, mainTop, mainBottom, raised, paper, meta)
	}

	footX := PadX
	footW := max(1, inner)
	if m.State.Searching {
		c.text(footX, footerY, "/", paper, surface, footW)
		if q := m.State.Query; q != "" {
			c.text(footX+1, footerY, q, meta, surface, max(1, footW-1))
		}
	} else if m.State.Feedback != "" {
		c.text(footX, footerY, m.State.Feedback, meta, surface, footW)
	} else {
		paintKeyLegend(c, footX, footerY, footW, m.footer(), paper, meta, surface)
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
	if compact {
		return "[ ↑↓ ] stack  [ ←→ ] layer  [ . ] copy  [ / ]  [ ? ]  [ q ]"
	}
	if m.Width <= 90 {
		return "[ ↑↓ ] stack  [ ←→ ] layer  [ o ] open  [ . ] copy  [ / ]  [ ? ]  [ q ]"
	}
	return "[ ↑↓ ] stack  [ ←→ ] layer  [ o ] open  [ . ] copy  [ a ] add  [ r ] refresh  [ / ] filter  [ esc ]  [ ? ]  [ q ]"
}

func helpItems() [][2]string {
	return [][2]string{
		{"↑↓", "stack"},
		{"←→", "layer"},
		{"o", "open"},
		{".", "copy"},
		{"/", "filter"},
		{"r", "refresh"},
		{"q", "quit"},
		{"?", "close"},
	}
}

func (m Model) paintHelp(c *canvas, top, bottom int, raised, paper, meta string) {
	items := helpItems()
	innerW := 0
	for _, item := range items {
		w := displayWidth("[ " + item[0] + " ] " + item[1])
		if w > innerW {
			innerW = w
		}
	}
	pad := 1
	w := innerW + 2 + pad*2
	h := len(items) + 2 + pad*2
	if maxW := innerWidth(m.Width); w > maxW {
		w = maxW
	}
	if maxH := max(3, bottom-top); h > maxH {
		h = maxH
	}
	x, y := PadX, top
	c.box(x, y, w, h, domain.Color("border"), raised, paper)
	cx := x + 1 + pad
	cy := y + 1 + pad
	cw := max(1, w-2-pad*2)
	row := 0
	for _, item := range items {
		if cy+row >= y+h-1-pad {
			break
		}
		key := "[ " + item[0] + " ]"
		c.text(cx, cy+row, key, paper, raised, cw)
		if displayWidth(key) < cw {
			c.text(cx+displayWidth(key), cy+row, " "+item[1], meta, raised, max(1, cw-displayWidth(key)))
		}
		row++
	}
}

func splitLegendItem(item string) (key, action string) {
	item = strings.TrimSpace(item)
	if strings.HasPrefix(item, "[") {
		if end := strings.Index(item, "]"); end >= 0 {
			return strings.TrimSpace(item[:end+1]), strings.TrimSpace(item[end+1:])
		}
	}
	key, action, _ = strings.Cut(item, " ")
	return key, action
}

func paintKeyLegend(c *canvas, x, y, maxWidth int, legend, paper, meta, bg string) {
	remain := maxWidth
	first := true
	for _, item := range strings.Split(legend, "  ") {
		if remain <= 0 {
			return
		}
		if !first {
			c.text(x, y, "  ", meta, bg, remain)
			n := min(2, remain)
			x += n
			remain -= n
		}
		first = false
		key, action := splitLegendItem(item)
		write := func(s, fg string) {
			if remain <= 0 || s == "" {
				return
			}
			c.text(x, y, s, fg, bg, remain)
			w := min(displayWidth(s), remain)
			x += w
			remain -= w
		}
		write(key, paper)
		if action != "" {
			write(" "+action, meta)
		}
	}
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
	gutter := layout.Gutter
	if gutter < 1 {
		gutter = ColGutter
	}
	statusW := layout.StatusWidth
	statusX := PadX + nameW + gutter + layout.BallsWidth + gutter
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

		ballX := PadX + nameW + gutter
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
			c.text(statusX, y, clip(stackHealth(stack), remain), stackHealthColor(stack), rowBg, remain)
		}
		y++
		if StackedInspector(m.Width) && selectedStack && m.State.CardVisible {
			inspH := m.stackedPaneHeight(listWidth)
			if y+inspH > bottom {
				inspH = max(1, bottom-y)
			}
			place := CardPlacement{Left: PadX, Top: y, Width: max(1, listWidth), Height: inspH, Compact: IsCompact(m.Width), Boxed: true}
			m.paintInspectorPane(c, place, surface, paper, meta)
			y += inspH
		}
	}
}

const inspectorLabelW = 10

type inspectorFact struct {
	label string
	value string
	fg    string
}

func inspectorFacts(pr domain.PullRequest) []inspectorFact {
	return []inspectorFact{
		{"status", inspectorStatus(pr), inspectorStatusColor(pr)},
		{"ci", inspectorCI(pr), inspectorCIColor(pr)},
		{"review", inspectorReview(pr), inspectorReviewColor(pr)},
		{"diff", fmt.Sprintf("+%d −%d", pr.Additions, pr.Deletions), domain.Color("paper")},
		{"branch", pr.Branch, domain.Color("paper")},
	}
}

func inspectorStatusColor(pr domain.PullRequest) string {
	return domain.Color(domain.StateColorToken(domain.GetDisplayState(pr)))
}

func inspectorCIColor(pr domain.PullRequest) string {
	switch pr.CI.State {
	case domain.CISuccess:
		return domain.Color("ready")
	case domain.CIFailure:
		return domain.Color("ciFailure")
	case domain.CIPending:
		return domain.Color("queued")
	default:
		return domain.Color("paper")
	}
}

func inspectorReviewColor(pr domain.PullRequest) string {
	if pr.Mergeable != nil && !*pr.Mergeable {
		return domain.Color("reviewBlocked")
	}
	if pr.ChangesRequested {
		return domain.Color("reviewBlocked")
	}
	if pr.Approvals > 0 {
		return domain.Color("ready")
	}
	return domain.Color("paper")
}

func inspectorStatus(pr domain.PullRequest) string {
	if label, ok := domain.DisplayStateLabel[domain.GetDisplayState(pr)]; ok {
		return label
	}
	return "pending"
}

func inspectorCI(pr domain.PullRequest) string {
	switch pr.CI.State {
	case domain.CISuccess:
		return "passed"
	case domain.CIFailure:
		return "failed"
	case domain.CIPending:
		return "pending"
	default:
		return "not reported"
	}
}

func inspectorReview(pr domain.PullRequest) string {
	if pr.Mergeable != nil && !*pr.Mergeable {
		return "merge conflict"
	}
	if pr.ChangesRequested {
		return "changes requested"
	}
	if pr.Approvals == 1 {
		return "1 approval"
	}
	if pr.Approvals > 1 {
		return itoa(pr.Approvals) + " approvals"
	}
	return "no decision"
}

func (m Model) paintInspectorPane(c *canvas, place CardPlacement, surface, paper, meta string) {
	x, y, w, h := place.Left, place.Top, place.Width, place.Height
	if w < 1 || h < 1 {
		return
	}
	bg := surface
	if place.Boxed {
		c.box(x, y, w, h, meta, surface, paper)
		x++
		y++
		w = max(0, w-2)
		h = max(0, h-2)
		if w < 1 || h < 1 {
			return
		}
		pad := 1
		if w > pad*2 && h > pad*2 {
			x += pad
			y += pad
			w -= pad * 2
			h -= pad * 2
		}
	} else {
		c.fill(x, y, w, h, surface)
	}
	pr, ok := m.SelectedPR()
	if !m.State.CardVisible || !ok {
		return
	}

	rowInk, rowRed := "", false
	if stack, ok := m.SelectedStack(); ok {
		rowInk, rowRed = rowDangerInk(stack)
	}

	maxLine := max(1, w)
	id := "#" + itoa(pr.Number)
	row := 0
	for _, line := range wrapWords(id+" "+pr.Title, maxLine) {
		if row >= h {
			return
		}
		c.text(x, y+row, line, paper, bg, maxLine)
		row++
	}
	if row >= h {
		return
	}
	row++
	for _, fact := range inspectorFacts(pr) {
		if row >= h {
			break
		}
		c.text(x, y+row, fact.label, meta, bg, min(inspectorLabelW, maxLine))
		if maxLine > inspectorLabelW && fact.value != "" {
			fg := fact.fg
			if rowRed {
				fg = rowInk
			}
			if fact.label == "diff" && !rowRed {
				paintDiff(c, x+inspectorLabelW, y+row, pr, maxLine-inspectorLabelW, bg)
			} else {
				c.text(x+inspectorLabelW, y+row, fact.value, fg, bg, maxLine-inspectorLabelW)
			}
		}
		row++
	}
}

func rowDangerInk(stack domain.Stack) (string, bool) {
	if len(stack.PRs) == 0 {
		return "", false
	}
	state := domain.GetDisplayState(stack.PRs[len(stack.PRs)-1])
	switch state {
	case domain.StateCIFailure, domain.StateReviewBlocked:
		return domain.Color(domain.StateColorToken(state)), true
	default:
		return "", false
	}
}

func paintDiff(c *canvas, x, y int, pr domain.PullRequest, maxW int, bg string) {
	plus := fmt.Sprintf("+%d", pr.Additions)
	minus := fmt.Sprintf("−%d", pr.Deletions)
	c.text(x, y, plus, domain.Color("ready"), bg, maxW)
	gap := displayWidth(plus) + 1
	if gap < maxW {
		c.text(x+gap, y, minus, domain.Color("ciFailure"), bg, maxW-gap)
	}
}

func stackHealthColor(stack domain.Stack) string {
	if len(stack.PRs) == 0 {
		return domain.Color("meta")
	}
	head := stack.PRs[len(stack.PRs)-1]
	return domain.Color(domain.StateColorToken(domain.GetDisplayState(head)))
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
		room = max(1, room-m.stackedPaneHeight(ListPaneWidth(m.Width)))
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
	border, pad := 1, 1
	innerW := max(1, listWidth-2*(border+pad))
	pr, ok := m.SelectedPR()
	if !ok {
		return 2*border + 2*pad + 1
	}
	title := "#" + itoa(pr.Number) + " " + pr.Title
	lines := len(wrapWords(title, innerW))
	return max(2*border+2*pad+1, 2*border+2*pad+lines+1+len(inspectorFacts(pr)))
}

func (m Model) ballHit(x, y int) (stackIndex, prIndex int, ok bool) {
	stacks := m.Stacks()
	if len(stacks) == 0 {
		return 0, 0, false
	}
	listWidth := ListPaneWidth(m.Width)
	if y < ListStartY || y >= ListBottomY(m.Height) || x < PadX || x >= PadX+listWidth {
		return 0, 0, false
	}
	sel := app.ClampSelection(m.State.Selection, stacks)
	start := m.listOrigin(len(stacks), sel.StackIndex, ListStartY, ListBottomY(m.Height))
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
	ballX := PadX + layout.NameWidth + layout.Gutter
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
