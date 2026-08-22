package tui_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/tui"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*(?:\x07|\x1b\\)`)

func strip(s string) string {
	return ansi.ReplaceAllString(s, "")
}

func ansiRGB(hex string) (int, int, int) {
	var r, g, b int
	fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

func ansiBG(hex string) string {
	r, g, b := ansiRGB(hex)
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

func ansiFG(hex string) string {
	r, g, b := ansiRGB(hex)
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

func makeUI(size tui.TerminalSize, storyID string) tui.Model {
	if storyID == "" {
		storyID = "mixed"
	}
	return tui.New(tui.Options{
		StoryID: storyID,
		Width:   size.Width,
		Height:  size.Height,
	})
}

func frameOf(m tui.Model) string {
	return strip(m.View())
}

func assertFits(t *testing.T, frame string, width int) {
	t.Helper()
	for i, line := range strings.Split(frame, "\n") {
		if utf8.RuneCountInString(line) > width {
			t.Fatalf("line %d is %d cells, wider than %d: %q", i, utf8.RuneCountInString(line), width, line)
		}
	}
}

func applyKey(m tui.Model, key tea.KeyMsg) tui.Model {
	next, _ := m.Update(key)
	return next.(tui.Model)
}

func key(s string) tea.KeyMsg {
	switch s {
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "home":
		return tea.KeyMsg{Type: tea.KeyHome}
	case "end":
		return tea.KeyMsg{Type: tea.KeyEnd}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	default:
		runes := []rune(s)
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: runes}
	}
}

func mouseMove(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonNone, Type: tea.MouseMotion}
}

func mousePress(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft, Type: tea.MouseLeft}
}

func TestHeaderCopyIsTwoLines(t *testing.T) {
	mixed := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"))
	lines := strings.Split(mixed, "\n")
	if len(lines) < 3 {
		t.Fatalf("short frame:\n%s", mixed)
	}
	if !strings.Contains(lines[1], "●-●-● DANGO") {
		t.Fatalf("line 1 must be ●-●-● DANGO:\n%s", mixed)
	}
	if strings.Contains(mixed, "🍡") {
		t.Fatalf("dumpling is gone:\n%s", mixed)
	}
	if strings.Contains(lines[1], "o-o-o") {
		t.Fatalf("logo must not use letter o:\n%s", mixed)
	}
	if strings.Contains(lines[1], "example/stacks") || strings.Contains(lines[1], "org/reponame") {
		t.Fatalf("line 1 must not carry the repo slug:\n%s", mixed)
	}
	if !strings.Contains(lines[1], "last fetched 2 mins ago") {
		t.Fatalf("line 1 must keep relative fetch age:\n%s", mixed)
	}
	if !strings.Contains(lines[2], "org/reponame  •  3 stacks / 8 layers") {
		t.Fatalf("line 2 must be repo and counts:\n%s", mixed)
	}
	if strings.Contains(mixed, "local deterministic data") {
		t.Fatalf("drop the data clause:\n%s", mixed)
	}

	chaos := makeUI(tui.TerminalSize{Width: 80, Height: 24}, "chaos")
	story := data.StoryByID("chaos")
	layers := 0
	for _, stack := range story.Stacks {
		layers += len(stack.PRs)
	}
	want := fmt.Sprintf("org/reponame  •  %d stacks / %d layers", len(story.Stacks), layers)
	frame := frameOf(chaos)
	if !strings.Contains(frame, want) {
		t.Fatalf("chaos counts: want %q\n%s", want, frame)
	}
	if strings.Contains(frame, "1 stacks / 20 layers") {
		t.Fatalf("chaos must not show the freight-only count:\n%s", frame)
	}
	if len(story.Stacks) != 300 {
		t.Fatalf("chaos fixture should be 300 stacks, got %d", len(story.Stacks))
	}
}

func TestDeterministicFramesAtCanonicalSizes(t *testing.T) {
	for _, size := range []tui.TerminalSize{{Width: 40, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 30}} {
		m := makeUI(size, "mixed")
		frame := frameOf(m)
		if !strings.Contains(frame, "●-●-● DANGO") {
			t.Fatalf("%dx%d missing brand mark:\n%s", size.Width, size.Height, frame)
		}
		if strings.Contains(frame, "🍡") {
			t.Fatalf("%dx%d still has the dumpling:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "org/reponame  •  3 stacks / 8 layers") {
			t.Fatalf("%dx%d missing story label:\n%s", size.Width, size.Height, frame)
		}
		if strings.Contains(frame, "local deterministic data") {
			t.Fatalf("%dx%d still has the old data clause:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "auth") {
			t.Fatalf("%dx%d missing stack:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "●") {
			t.Fatalf("%dx%d missing balls:\n%s", size.Width, size.Height, frame)
		}
		assertFits(t, frame, size.Width)
	}
}

func TestKeyboardAndHoverRevealTheSameInspector(t *testing.T) {
	size := tui.TerminalSize{Width: 80, Height: 24}
	m := applyKey(makeUI(size, "mixed"), key("right"))
	keyboard := frameOf(m)
	if !strings.Contains(keyboard, "#185 Keep service identity") {
		t.Fatalf("keyboard inspector:\n%s", keyboard)
	}
	if !strings.Contains(keyboard, "○-●-○") {
		t.Fatalf("keyboard selection glyph:\n%s", keyboard)
	}

	first := data.FixtureStories[0].Stacks[0]
	point := tui.GetBallPoint(size, 0, 1, len(first.PRs))
	hovered, _ := makeUI(size, "mixed").Update(mouseMove(point.X, point.Y))
	hover := frameOf(hovered.(tui.Model))
	if !strings.Contains(hover, "#185 Keep service identity") {
		t.Fatalf("hover inspector:\n%s", hover)
	}
	if !strings.Contains(hover, "ready to merge") {
		t.Fatalf("hover headline:\n%s", hover)
	}
}

func TestCompactCardAndHomeEnd(t *testing.T) {
	size := tui.TerminalSize{Width: 40, Height: 20}
	m := applyKey(makeUI(size, "mixed"), key("end"))
	if !strings.Contains(frameOf(m), "#241") {
		t.Fatalf("end should jump to the last stack:\n%s", frameOf(m))
	}
	m = applyKey(m, key("home"))
	frame := frameOf(m)
	if !strings.Contains(frame, "#184") {
		t.Fatalf("home should jump to the first stack:\n%s", frame)
	}
	if !strings.Contains(frame, "status") || !strings.Contains(frame, "branch") {
		t.Fatalf("compact inspector should be labeled rows:\n%s", frame)
	}
	if !strings.Contains(frame, "[ ↑↓ ] stack") || !strings.Contains(frame, "[ ←→ ] layer") {
		t.Fatalf("compact footer:\n%s", frame)
	}
	assertFits(t, frame, 40)
}

func TestBallHitCellsSelect(t *testing.T) {
	size := tui.TerminalSize{Width: 80, Height: 24}
	first := data.FixtureStories[0].Stacks[0]
	point := tui.GetBallPoint(size, 0, 1, len(first.PRs))
	m, _ := makeUI(size, "mixed").Update(mousePress(point.X+1, point.Y))
	frame := frameOf(m.(tui.Model))
	if !strings.Contains(frame, "#185 Keep service identity") {
		t.Fatalf("click should select:\n%s", frame)
	}
	if strings.Contains(frame, "Checked out") {
		t.Fatalf("click must not checkout:\n%s", frame)
	}
	head := tui.GetBallPoint(size, 0, 2, len(first.PRs))
	m, _ = m.Update(mousePress(head.X+1, head.Y))
	frame = frameOf(m.(tui.Model))
	if !strings.Contains(frame, "#186 Remove implicit session fallback") {
		t.Fatalf("head trailing cell click:\n%s", frame)
	}
	if strings.Contains(frame, "Checked out") {
		t.Fatalf("click must not checkout:\n%s", frame)
	}
	if tui.GetRowLayout(80, len(first.PRs)).BallsWidth != tui.BallColW {
		t.Fatal("ball column should stay locked")
	}
}

func TestLocalFilter(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("/"))
	for _, r := range "composer" {
		m = applyKey(m, key(string(r)))
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "composer tokens") {
		t.Fatalf("filtered stack missing:\n%s", frame)
	}
	if strings.Contains(frame, "auth cleanup") {
		t.Fatalf("unfiltered stack still visible:\n%s", frame)
	}
}

func TestFixtureCacheAndEmptyStates(t *testing.T) {
	stale := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "draft"))
	if !strings.Contains(stale, "release notes") {
		t.Fatalf("stale:\n%s", stale)
	}
	if strings.Contains(stale, "fixture cache ·") {
		t.Fatalf("footer must not be a middot status sentence:\n%s", stale)
	}
	empty := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "all-merged"))
	if !strings.Contains(empty, "No open stacks in this fixture") {
		t.Fatalf("empty:\n%s", empty)
	}
	errFrame := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "ci-failing"))
	if !strings.Contains(errFrame, "Refresh failed in this fixture.") {
		t.Fatalf("error empty:\n%s", errFrame)
	}
	if strings.Contains(errFrame, "fixture refresh failed · no cached stacks") {
		t.Fatalf("error status leaked into the footer strip:\n%s", errFrame)
	}
}

func TestSearchOwnsFooterAndQIsQuery(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("/"))
	searching := frameOf(m)
	if !strings.Contains(searching, "/") {
		t.Fatalf("search field:\n%s", searching)
	}
	if strings.Contains(searching, "type to filter") {
		t.Fatalf("search should not be a help modal:\n%s", searching)
	}
	if strings.Contains(searching, "q quit") {
		t.Fatalf("q quit leaked into search:\n%s", searching)
	}
	m = applyKey(m, key("q"))
	if m.View() == "" {
		t.Fatal("q should not quit while searching")
	}
	if !strings.Contains(frameOf(m), "q") {
		t.Fatalf("q should become query text:\n%s", frameOf(m))
	}
}

func TestEightyColumnFooterAndFocus(t *testing.T) {
	m := makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed")
	raw := m.View()
	frame := strip(raw)
	if !strings.Contains(frame, "●") {
		t.Fatalf("focused layer:\n%s", frame)
	}
	if !strings.Contains(frame, "[ ↑↓ ] stack") || !strings.Contains(frame, "[ o ] open") || !strings.Contains(frame, "[ . ] copy") {
		t.Fatalf("80-col footer should be a key strip:\n%s", frame)
	}
	if strings.Contains(frame, "[ enter ]") {
		t.Fatalf("enter must leave the footer:\n%s", frame)
	}
	if strings.Contains(frame, "fixture cache ·") || strings.Contains(frame, " · ") && strings.Contains(frame, "q quit") {
		t.Fatalf("footer must not be a middot sentence:\n%s", frame)
	}
	if !strings.Contains(raw, "38;2;") {
		t.Fatal("expected truecolor OKLCH palette in the fixture frame")
	}
}

func TestResizeAndCardClamp(t *testing.T) {
	m := makeUI(tui.TerminalSize{Width: 160, Height: 40}, "mixed")
	for _, size := range []tui.TerminalSize{
		{Width: 40, Height: 20},
		{Width: 80, Height: 24},
		{Width: 120, Height: 30},
		{Width: 160, Height: 40},
	} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size.Width, Height: size.Height})
		m = next.(tui.Model)
		frame := frameOf(m)
		if !strings.Contains(frame, "●-●-● DANGO") {
			t.Fatalf("%dx%d missing brand mark:\n%s", size.Width, size.Height, frame)
		}
		if strings.Contains(frame, "🍡") {
			t.Fatalf("%dx%d still has the dumpling:\n%s", size.Width, size.Height, frame)
		}
		assertFits(t, frame, size.Width)
		placement := tui.GetInspectorSize(size)
		if placement.Left < tui.PadX || placement.Top < 1 {
			t.Fatalf("inspector underflow at %dx%d: %+v", size.Width, size.Height, placement)
		}
		if placement.Left+placement.Width > size.Width-tui.PadX {
			t.Fatalf("inspector overflows right at %dx%d: %+v", size.Width, size.Height, placement)
		}
		if placement.Top+placement.Height > tui.ListBottomY(size.Height) {
			t.Fatalf("inspector overflows into footer air at %dx%d: %+v", size.Width, size.Height, placement)
		}
	}
	large := data.StoryByID("large-stack").Stacks[0]
	compact := tui.GetRowLayout(40, len(large.PRs))
	if !compact.Compact {
		t.Fatal("40-col large stack should be compact")
	}
	if compact.NameWidth+compact.BallsWidth > 38 {
		t.Fatalf("compact row too wide: name=%d balls=%d", compact.NameWidth, compact.BallsWidth)
	}
}

func TestQQuits(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("q"))
	if m.View() != "" {
		t.Fatalf("quit should clear the view, got %q", m.View())
	}
}

func TestSimulatedActionsStayHonest(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("enter"))
	if strings.Contains(frameOf(m), "Checked out") {
		t.Fatalf("enter must not checkout:\n%s", frameOf(m))
	}
	m = applyKey(m, key("o"))
	if !strings.Contains(frameOf(m), "Opening https://github.com/example/stacks/pull/184") {
		t.Fatalf("open:\n%s", frameOf(m))
	}
	m = applyKey(m, key("r"))
	if !strings.Contains(frameOf(m), "⠋") {
		t.Fatalf("refresh:\n%s", frameOf(m))
	}
}

func TestInspectorIsARightColumn(t *testing.T) {
	size := tui.TerminalSize{Width: 120, Height: 30}
	m := makeUI(size, "mixed")
	raw := m.View()
	wide := strip(raw)
	if !strings.Contains(wide, "#184 Split auth scope from session checks") {
		t.Fatalf("inspector missing title:\n%s", wide)
	}
	if strings.Contains(wide, "┌") || strings.Contains(wide, "└") {
		t.Fatalf("inspector must be a pane, not a postcard:\n%s", wide)
	}
	if !strings.Contains(wide, "│") {
		t.Fatalf("inspector pane needs one vertical rule:\n%s", wide)
	}
	lines := strings.Split(wide, "\n")
	if len(lines) < 7 {
		t.Fatalf("short frame:\n%s", wide)
	}
	if strings.TrimSpace(lines[0]) != "" {
		t.Fatalf("expected one blank row at the top:\n%s", wide)
	}
	if strings.TrimSpace(lines[3]) != "" {
		t.Fatalf("expected one blank row under the header:\n%s", wide)
	}
	if strings.TrimSpace(lines[4]) == "" {
		t.Fatalf("list should start after one header air row:\n%s", wide)
	}
	if strings.TrimSpace(lines[len(lines)-2]) != "" {
		t.Fatalf("expected one blank row over the footer:\n%s", wide)
	}
	place := tui.GetInspectorSize(size)
	if place.Left < 60 {
		t.Fatalf("inspector should be a right-hand column, left=%d", place.Left)
	}
	fieldBG := ansiBG(domain.Color("surface"))
	if !strings.Contains(raw, fieldBG) {
		t.Fatalf("list field missing from frame: %s", fieldBG)
	}
	if strings.Contains(raw, ansiBG("#d9c8aa")) {
		t.Fatal("tan postcard fill should be gone")
	}
}

func TestInspectorIsLabeledRows(t *testing.T) {
	size := tui.TerminalSize{Width: 120, Height: 30}
	frame := frameOf(makeUI(size, "mixed"))
	if !strings.Contains(frame, "#184 Split auth scope from session checks") {
		t.Fatalf("paper title missing:\n%s", frame)
	}
	if strings.Contains(frame, "split auth scope from session checks, keep") {
		t.Fatalf("inspector must not dump a comma layer list:\n%s", frame)
	}
	needles := []string{
		"status    merged",
		"ci        not reported",
		"review    no decision",
		"diff      +43 −12",
		"branch    gm/stacks-184",
	}
	for _, needle := range needles {
		if !strings.Contains(frame, needle) {
			t.Fatalf("inspector missing %q:\n%s", needle, frame)
		}
	}
	lines := strings.Split(frame, "\n")
	titleAt := -1
	pane := func(line string) string {
		if idx := strings.Index(line, "│"); idx >= 0 {
			return line[idx+len("│"):]
		}
		return line
	}
	for i, line := range lines {
		if strings.Contains(pane(line), "#184 Split auth scope from session checks") {
			titleAt = i
			break
		}
	}
	if titleAt < 0 || titleAt+2 >= len(lines) {
		t.Fatalf("title row missing:\n%s", frame)
	}
	if strings.TrimSpace(pane(lines[titleAt+1])) != "" {
		t.Fatalf("expected one blank row under the title:\n%s", frame)
	}
	if !strings.Contains(pane(lines[titleAt+2]), "status") {
		t.Fatalf("first fact should be status:\n%s", frame)
	}
	if strings.Contains(frame, "┌") || strings.Contains(frame, "└") {
		t.Fatalf("inspector must not invent a box:\n%s", frame)
	}
}

func TestFooterKeysAreBracketed(t *testing.T) {
	for _, size := range []tui.TerminalSize{{Width: 40, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 30}} {
		frame := frameOf(makeUI(size, "mixed"))
		lines := strings.Split(frame, "\n")
		footer := lines[len(lines)-1]
		if !strings.Contains(footer, "[ ↑↓ ]") || !strings.Contains(footer, "stack") {
			t.Fatalf("%dx%d footer missing bracketed keys:\n%s", size.Width, size.Height, footer)
		}
		if strings.Contains(footer, "↑↓ stack") && !strings.Contains(footer, "[ ↑↓ ] stack") {
			t.Fatalf("%dx%d key is not bracketed:\n%s", size.Width, size.Height, footer)
		}
	}
}

func TestListColumnsHaveGutters(t *testing.T) {
	size := tui.TerminalSize{Width: 120, Height: 30}
	m := makeUI(size, "freight")
	frame := frameOf(m)
	listWidth := tui.ListPaneWidth(size.Width)
	layout := tui.GetListRowLayout(listWidth, size.Width, 20)
	if layout.Gutter < tui.ColGutter {
		t.Fatalf("gutter too small: %d", layout.Gutter)
	}
	if layout.StatusWidth != 0 {
		t.Fatalf("status is not a list column, StatusWidth=%d", layout.StatusWidth)
	}
	nameEnd := tui.PadX + layout.NameWidth
	ballStart := nameEnd + layout.Gutter
	var row string
	for _, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, "freight train") {
			row = line
			break
		}
	}
	if row == "" {
		t.Fatalf("freight row missing:\n%s", frame)
	}
	listPart := row
	if idx := strings.Index(row, "│"); idx >= 0 {
		listPart = row[:idx]
	}
	runes := []rune(listPart)
	if ballStart >= len(runes) {
		t.Fatalf("row shorter than ball column: %q", listPart)
	}
	for x := nameEnd; x < ballStart; x++ {
		if runes[x] != ' ' {
			t.Fatalf("name/balls gutter smeared at %d: %q", x, listPart)
		}
	}
	for _, word := range []string{"pending", "ready", "blocked", "ci failed", "queued", "draft", "merged"} {
		if strings.Contains(listPart, word) {
			t.Fatalf("list row must not carry status words %q: %q", word, listPart)
		}
	}
	if tui.GetRowLayout(80, 3).Gutter < tui.ColGutter {
		t.Fatal("do not invent drag-resize; keep a fixed gutter")
	}
}

func TestHoverFillsBallAndShowsInspector(t *testing.T) {
	size := tui.TerminalSize{Width: 80, Height: 24}
	first := data.FixtureStories[0].Stacks[0]
	point := tui.GetBallPoint(size, 0, 1, len(first.PRs))
	m, _ := makeUI(size, "mixed").Update(mouseMove(point.X, point.Y))
	frame := frameOf(m.(tui.Model))
	if !strings.Contains(frame, "┌") || !strings.Contains(frame, "└") {
		t.Fatalf("stacked hover card needs one inset box:\n%s", frame)
	}
	if !strings.Contains(frame, "#185 Keep service identity") {
		t.Fatalf("hover inspector:\n%s", frame)
	}
	if !strings.Contains(frame, "○-●-○") {
		t.Fatalf("hover should fill the focused ball:\n%s", frame)
	}
}

func TestTypeIsThreeInks(t *testing.T) {
	raw := makeUI(tui.TerminalSize{Width: 120, Height: 30}, "mixed").View()
	paper := ansiFG(domain.Color("paper"))
	meta := ansiFG(domain.Color("meta"))
	failed := ansiFG(domain.Color("ciFailure"))
	if !strings.Contains(raw, paper) {
		t.Fatal("selected stack name and PR id must use paper ink")
	}
	if !strings.Contains(raw, meta) {
		t.Fatal("everything else must use meta")
	}
	if !strings.Contains(raw, failed) {
		t.Fatal("failed / ready / blocked must keep their status colors")
	}
	frame := strip(raw)
	if !strings.Contains(frame, "[ ↑↓ ]") || !strings.Contains(frame, "stack") {
		t.Fatalf("footer key legend missing:\n%s", frame)
	}
	if strings.Contains(frame, "fixture cache ·") {
		t.Fatalf("idle footer is still a middot status sentence:\n%s", frame)
	}
	if strings.Contains(frame, "ci failed") {
		t.Fatalf("status words must leave the list; keep them on balls and in the inspector:\n%s", frame)
	}
}

func TestHelpOverlayToggles(t *testing.T) {
	m := makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed")
	idle := frameOf(m)
	if strings.Contains(idle, "[ ? ] close") {
		t.Fatalf("help should start closed:\n%s", idle)
	}
	m = applyKey(m, key("?"))
	frame := frameOf(m)
	for _, needle := range []string{
		"[ ↑↓ ] stack",
		"[ ←→ ] layer",
		"[ o ] open",
		"[ . ] copy",
		"[ / ] filter",
		"[ r ] refresh",
		"[ q ] quit",
		"[ ? ] close",
	} {
		if !strings.Contains(frame, needle) {
			t.Fatalf("help overlay missing %q:\n%s", needle, frame)
		}
	}
	if strings.Contains(frame, "[ enter ]") {
		t.Fatalf("help must not invent enter:\n%s", frame)
	}
	m = applyKey(m, key("?"))
	closed := frameOf(m)
	if strings.Contains(closed, "[ ? ] close") {
		t.Fatalf("? should close help:\n%s", closed)
	}
}

func TestStackedCardIsInsetBox(t *testing.T) {
	stacked := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"))
	if !strings.Contains(stacked, "┌") || !strings.Contains(stacked, "└") {
		t.Fatalf("stacked card needs one dim box:\n%s", stacked)
	}
	if !strings.Contains(stacked, "│ #184") && !strings.Contains(stacked, "│  #184") {
		t.Fatalf("stacked card needs padding inside the box:\n%s", stacked)
	}
	wide := frameOf(makeUI(tui.TerminalSize{Width: 120, Height: 30}, "mixed"))
	if strings.Contains(wide, "┌") || strings.Contains(wide, "└") {
		t.Fatalf("wide pane must stay borderless:\n%s", wide)
	}
}

func TestMarkUsesThreeDistinctLogoInks(t *testing.T) {
	m := makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed")
	seen := map[string]bool{}
	for _, token := range m.LogoDots {
		if !domain.IsLogoToken(token) {
			t.Fatalf("mark ink %q is not in the seven", token)
		}
		if seen[token] {
			t.Fatalf("mark inks are not distinct: %v", m.LogoDots)
		}
		seen[token] = true
	}
	if len(seen) != 3 {
		t.Fatalf("need three distinct mark inks, got %v", m.LogoDots)
	}
	frame := strip(m.View())
	if !strings.Contains(frame, "●-●-● DANGO") {
		t.Fatalf("packed ASCII mark missing:\n%s", frame)
	}
	if strings.Contains(frame, "🍡") {
		t.Fatal("dumpling must not be painted")
	}
}

func TestFooterHasNoEnter(t *testing.T) {
	for _, size := range []tui.TerminalSize{{Width: 40, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 30}} {
		frame := frameOf(makeUI(size, "mixed"))
		if strings.Contains(frame, "[ enter ]") || strings.Contains(frame, "checkout") {
			t.Fatalf("%dx%d still advertises enter/checkout:\n%s", size.Width, size.Height, frame)
		}
	}
}
