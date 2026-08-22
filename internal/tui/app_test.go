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

func TestDeterministicFramesAtCanonicalSizes(t *testing.T) {
	for _, size := range []tui.TerminalSize{{Width: 40, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 30}} {
		m := makeUI(size, "mixed")
		frame := frameOf(m)
		if !strings.Contains(frame, "●-●-● DANGO") {
			t.Fatalf("%dx%d missing brand mark:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "3 stacks / 8 layers") {
			t.Fatalf("%dx%d missing story label:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "auth clean") {
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
	if !strings.Contains(frame, "o open") {
		t.Fatalf("compact card hint:\n%s", frame)
	}
	if !strings.Contains(frame, "↑↓ stack  ←→ layer  /  ?  q") {
		t.Fatalf("compact footer:\n%s", frame)
	}
	assertFits(t, frame, 40)
}

func TestBallHitCellsCheckout(t *testing.T) {
	size := tui.TerminalSize{Width: 80, Height: 24}
	first := data.FixtureStories[0].Stacks[0]
	point := tui.GetBallPoint(size, 0, 1, len(first.PRs))
	m, _ := makeUI(size, "mixed").Update(mousePress(point.X+1, point.Y))
	if !strings.Contains(frameOf(m.(tui.Model)), "Checked out gm/stacks-185 · fixture simulation") {
		t.Fatalf("connector click:\n%s", frameOf(m.(tui.Model)))
	}
	head := tui.GetBallPoint(size, 0, 2, len(first.PRs))
	m, _ = m.Update(mousePress(head.X+1, head.Y))
	if !strings.Contains(frameOf(m.(tui.Model)), "Checked out gm/stacks-186 · fixture simulation") {
		t.Fatalf("head trailing cell click:\n%s", frameOf(m.(tui.Model)))
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
	if !strings.Contains(strings.Join(strings.Fields(searching), " "), "type to filter backspace edits esc clears / exits") {
		t.Fatalf("search footer:\n%s", searching)
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
	if !strings.Contains(frame, "↑↓ stack  ←→ layer") {
		t.Fatalf("80-col footer should be a key strip:\n%s", frame)
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
		assertFits(t, frame, size.Width)
		placement := tui.GetInspectorSize(size)
		if placement.Left < tui.PadX || placement.Top < 1 {
			t.Fatalf("inspector underflow at %dx%d: %+v", size.Width, size.Height, placement)
		}
		if placement.Left+placement.Width > size.Width-tui.PadX {
			t.Fatalf("inspector overflows right at %dx%d: %+v", size.Width, size.Height, placement)
		}
		if placement.Top+placement.Height > size.Height-1 {
			t.Fatalf("inspector overflows bottom at %dx%d: %+v", size.Width, size.Height, placement)
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
	if !strings.Contains(frameOf(m), "Checked out gm/stacks-184 · fixture simulation") {
		t.Fatalf("checkout:\n%s", frameOf(m))
	}
	m = applyKey(m, key("o"))
	if !strings.Contains(frameOf(m), "Opening https://github.com/example/stacks/pull/184") {
		t.Fatalf("open:\n%s", frameOf(m))
	}
	m = applyKey(m, key("r"))
	if !strings.Contains(frameOf(m), "Fixture data refreshed · no network") {
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
	if strings.TrimSpace(lines[3]) != "" || strings.TrimSpace(lines[4]) != "" {
		t.Fatalf("expected two blank rows after the header:\n%s", wide)
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

func TestInspectorShowsOneSummaryLine(t *testing.T) {
	size := tui.TerminalSize{Width: 120, Height: 30}
	frame := frameOf(makeUI(size, "mixed"))
	if !strings.Contains(frame, "#184 Split auth scope from session checks") {
		t.Fatalf("paper title missing:\n%s", frame)
	}
	hits := 0
	for _, line := range strings.Split(frame, "\n") {
		if strings.Contains(line, "split auth scope") {
			hits++
		}
	}
	if hits != 1 {
		t.Fatalf("summary must be one meta line under the title, got %d:\n%s", hits, frame)
	}
	if strings.Contains(frame, "┌") || strings.Contains(frame, "└") {
		t.Fatalf("summary must not invent a box:\n%s", frame)
	}
}

func TestHoverFillsBallAndShowsInspector(t *testing.T) {
	size := tui.TerminalSize{Width: 80, Height: 24}
	first := data.FixtureStories[0].Stacks[0]
	point := tui.GetBallPoint(size, 0, 1, len(first.PRs))
	m, _ := makeUI(size, "mixed").Update(mouseMove(point.X, point.Y))
	frame := frameOf(m.(tui.Model))
	if strings.Contains(frame, "┌") {
		t.Fatalf("hover must not drop a postcard on the list:\n%s", frame)
	}
	if !strings.Contains(frame, "#185 Keep service identity") {
		t.Fatalf("hover inspector:\n%s", frame)
	}
	if !strings.Contains(frame, "○-●-○") {
		t.Fatalf("hover should fill the focused ball:\n%s", frame)
	}
}

func TestTypeIsPaperOrMeta(t *testing.T) {
	raw := makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed").View()
	paper := ansiFG(domain.Color("paper"))
	meta := ansiFG(domain.Color("meta"))
	if !strings.Contains(raw, paper) {
		t.Fatal("selected stack name / inspector title must use paper ink")
	}
	if !strings.Contains(raw, meta) {
		t.Fatal("everything else must use meta")
	}
	frame := strip(raw)
	if strings.Contains(frame, "fixture cache ·") {
		t.Fatalf("idle footer is still a middot status sentence:\n%s", frame)
	}
}
