package tui

import (
	"fmt"
	"strings"

	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/live"
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

	if m.splash() {
		m.paintSplash(c, width, height, surface, paper, meta, stick)
		footX := PadX
		footW := max(1, inner)
		footerY := height - 1
		if m.State.Feedback != "" {
			c.text(footX, footerY, m.State.Feedback, meta, surface, footW)
		} else if m.fetchErr != nil {
			paintKeyLegend(c, footX, footerY, footW, errorFooter(), paper, meta, surface)
		}
		return c.render()
	}

	// y=0 is the one blank row at the top.
	m.paintBrand(c, width, surface, paper, meta, stick)
	// one blank row under the header, then the list.

	footerY := height - 1
	mainTop := ListStartY
	mainBottom := ListBottomY(height)
	if mainBottom < mainTop {
		mainBottom = mainTop
	}

	if m.showError() {
		m.paintError(c, width, mainTop, mainBottom, surface, raised, paper)
	} else if StackedInspector(width) {
		m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, paper, meta, stick)
	} else {
		m.paintList(c, listWidth, mainTop, mainBottom, surface, raised, paper, meta, stick)
		m.paintRule(c, mainTop, mainBottom, meta, surface)
		m.paintInspectorPane(c, insp, surface, paper, meta)
	}
	if m.Help && !m.showError() {
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
		legend := m.footer()
		if m.showError() {
			legend = errorFooter()
		}
		paintKeyLegend(c, footX, footerY, footW, legend, paper, meta, surface)
	}

	return c.render()
}

func (m Model) paintError(c *canvas, width, top, bottom int, surface, raised, paper string) {
	maxW := innerWidth(width)
	y := top
	if y >= bottom {
		return
	}
	c.text(PadX, y, "Could not fetch pull requests.", paper, surface, maxW)
	y += 2
	for _, line := range wrapWords(m.errorCopyText(), maxW) {
		if y >= bottom {
			break
		}
		c.fill(PadX, y, maxW, 1, raised)
		c.text(PadX, y, line, paper, raised, maxW)
		y++
	}
}

func errorFooter() string {
	return "[ . ] copy  [ q ]"
}

func (m Model) paintBrand(c *canvas, width int, surface, paper, meta, stick string) {
	badge := m.fetchBadge()
	c.text(width-PadX-displayWidth(badge), PadTop, badge, meta, surface, displayWidth(badge))
	x := PadX
	for i, token := range m.LogoDots {
		if token == "" || !domain.IsLogoToken(token) {
			token = domain.LogoTokens[i]
		}
		c.set(x, PadTop, '●', domain.Color(token), surface)
		if i < 2 {
			c.set(x+1, PadTop, '-', stick, surface)
		}
		x += 2
	}
	c.text(PadX+6, PadTop, "DANGO", paper, surface, displayWidth("DANGO"))
	line2 := repoCountLine(m.repoLabel(), m.stackCount(), m.layerCount())
	if m.waiting() {
		line2 = "fetching " + m.repoLabel()
	}
	c.text(PadX, PadTop+1, line2, meta, surface, innerWidth(width))
}

func repoCountLine(repo string, stacks, layers int) string {
	return repo + "  •  " + itoa(stacks) + " stacks / " + itoa(layers) + " layers"
}

func stackListName(stack domain.Stack) string {
	if name := strings.TrimSpace(stack.Name); name != "" {
		return name
	}
	return live.GhTitle(stack)
}

func layerBallInk(pr domain.PullRequest) string {
	return domain.Color(domain.StateColorToken(domain.GetDisplayState(pr)))
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
		width := max(1, listWidth)
		y := top
		for _, line := range wrapWords(m.emptyMessage(), width) {
			if y >= bottom {
				break
			}
			c.text(PadX, y, line, meta, surface, width)
			y++
		}
		return
	}
	sel := app.ClampSelection(m.State.Selection, stacks)
	layout := GetListRowLayout(listWidth, m.Width, 0)
	nameW := layout.NameWidth
	gutter := layout.Gutter
	if gutter < 1 {
		gutter = ColGutter
	}
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
		c.text(PadX, y, marker+stackListName(stack), nameFg, rowBg, nameW)

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
				fg := layerBallInk(pr)
				selected := selectedStack && cell.pr == sel.PRIndex
				glyph := '○'
				if selected {
					glyph = '●'
				}
				c.set(x, y, glyph, fg, rowBg)
				x++
			} else {
				pr := stack.PRs[cell.pr]
				fg := layerBallInk(pr)
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

type inspectorPart struct {
	text string
	fg   string
}

type inspectorFact struct {
	label string
	value string
	fg    string
	parts []inspectorPart
}

func inspectorFacts(pr domain.PullRequest) []inspectorFact {
	paper := domain.Color("paper")
	meta := domain.Color("meta")
	return []inspectorFact{
		{"status", inspectorStatus(pr), inspectorStatusColor(pr), nil},
		{"ci", inspectorCI(pr), paper, nil},
		{"review", inspectorReview(pr), paper, nil},
		{"diff", fmt.Sprintf("+%d −%d", pr.Additions, pr.Deletions), paper, nil},
		{"branch", pr.Branch, paper, nil},
		inspectorLabelsFact(pr, meta),
		inspectorAuthorFact(pr, paper, meta),
	}
}

func inspectorLabelsFact(pr domain.PullRequest, meta string) inspectorFact {
	if len(pr.Labels) == 0 {
		return inspectorFact{label: "labels", value: "none", fg: meta, parts: []inspectorPart{{text: "none", fg: meta}}}
	}
	var parts []inspectorPart
	var names []string
	for i, lab := range pr.Labels {
		if i > 0 {
			parts = append(parts, inspectorPart{text: " ", fg: meta})
		}
		fg := domain.NormalizeHex(lab.Color)
		if fg == "" {
			fg = meta
		}
		parts = append(parts, inspectorPart{text: lab.Name, fg: fg})
		names = append(names, lab.Name)
	}
	return inspectorFact{label: "labels", value: strings.Join(names, " "), fg: meta, parts: parts}
}

func inspectorAuthorFact(pr domain.PullRequest, paper, meta string) inspectorFact {
	login := strings.TrimSpace(pr.Author)
	if login == "" {
		return inspectorFact{label: "author", value: "none", fg: meta, parts: []inspectorPart{{text: "none", fg: meta}}}
	}
	ink := pr.AuthorColor
	if ink == "" {
		ink = domain.LoginColor(login)
	}
	return inspectorFact{
		label: "author",
		value: "● " + login,
		fg:    paper,
		parts: []inspectorPart{
			{text: "●", fg: ink},
			{text: " " + login, fg: paper},
		},
	}
}

func inspectorStatusColor(pr domain.PullRequest) string {
	return domain.Color(domain.StateColorToken(domain.GetDisplayState(pr)))
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
	row = m.paintStackDescription(c, x, y, row, h, maxLine, meta, bg)
	if row >= h {
		return
	}
	for _, fact := range inspectorFacts(pr) {
		if row >= h {
			break
		}
		c.text(x, y+row, fact.label, meta, bg, min(inspectorLabelW, maxLine))
		if maxLine > inspectorLabelW && (fact.value != "" || len(fact.parts) > 0) {
			paintFactValue(c, x+inspectorLabelW, y+row, fact, pr, maxLine-inspectorLabelW, bg)
		}
		row++
	}
}

// inspectorDescLines is the reserved body of the inspector description
// slot. A landed description paints into these rows; it must not grow
// the card or shift the fact rows.
const inspectorDescLines = 2

func (m Model) reserveInspectorDesc() bool {
	return m.Live && !m.Provider.Empty()
}

func (m Model) inspectorDescRows(innerW int) int {
	if m.reserveInspectorDesc() {
		return inspectorDescLines + 1
	}
	if !m.Live && !m.File {
		return 0
	}
	stack, ok := m.SelectedStack()
	if !ok {
		return 0
	}
	desc := strings.TrimSpace(stack.Description)
	if desc == "" {
		return 0
	}
	return len(wrapDesc(desc, max(1, innerW))) + 1
}

func (m Model) paintStackDescription(c *canvas, x, y, row, h, maxLine int, meta, bg string) int {
	if m.reserveInspectorDesc() {
		desc := ""
		if stack, ok := m.SelectedStack(); ok {
			desc = strings.TrimSpace(stack.Description)
		}
		painted := 0
		if desc != "" {
			for _, line := range wrapDesc(desc, maxLine) {
				if painted >= inspectorDescLines || row >= h {
					break
				}
				c.text(x, y+row, line, meta, bg, maxLine)
				row++
				painted++
			}
		}
		row += inspectorDescLines - painted
		if row < h {
			row++
		}
		return row
	}
	if !m.Live && !m.File {
		return row
	}
	stack, ok := m.SelectedStack()
	if !ok {
		return row
	}
	desc := strings.TrimSpace(stack.Description)
	if desc == "" {
		return row
	}
	for _, line := range wrapDesc(desc, maxLine) {
		if row >= h {
			return row
		}
		c.text(x, y+row, line, meta, bg, maxLine)
		row++
	}
	if row < h {
		row++
	}
	return row
}

func wrapDesc(desc string, width int) []string {
	lines := wrapWords(desc, width)
	if len(lines) > inspectorDescLines {
		return lines[:inspectorDescLines]
	}
	return lines
}

func paintFactValue(c *canvas, x, y int, fact inspectorFact, pr domain.PullRequest, maxW int, bg string) {
	if fact.label == "diff" {
		paintDiff(c, x, y, pr, maxW, bg)
		return
	}
	if len(fact.parts) > 0 {
		cx, remain := x, maxW
		for _, part := range fact.parts {
			if remain <= 0 {
				return
			}
			c.text(cx, y, part.text, part.fg, bg, remain)
			w := displayWidth(part.text)
			cx += w
			remain -= w
		}
		return
	}
	c.text(x, y, fact.value, fact.fg, bg, maxW)
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
	descLines := m.inspectorDescRows(innerW)
	return max(2*border+2*pad+1, 2*border+2*pad+lines+1+descLines+len(inspectorFacts(pr)))
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
