package tui_test

import (
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/tui"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*(?:\x07|\x1b\\)`)

func strip(s string) string {
	return ansi.ReplaceAllString(s, "")
}

func makeUI(size tui.TerminalSize, storyID string) tui.Model {
	if storyID == "" {
		storyID = "mixed"
	}
	return tui.New(tui.Options{
		Mode:    tui.ModeStories,
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
		if !strings.Contains(frame, "STACKS UI LAB") {
			t.Fatalf("%dx%d missing title:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "mixed health") {
			t.Fatalf("%dx%d missing story label:\n%s", size.Width, size.Height, frame)
		}
		if !strings.Contains(frame, "auth cleanup") {
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
	if !strings.Contains(keyboard, "#185 Keep service identity explicit") {
		t.Fatalf("keyboard inspector:\n%s", keyboard)
	}
	if !strings.Contains(keyboard, "●—◉—●") {
		t.Fatalf("keyboard selection glyph:\n%s", keyboard)
	}

	first := data.FixtureStories[0].Stacks[0]
	point := tui.GetBallPoint(size, 0, 1, len(first.PRs))
	hovered, _ := makeUI(size, "mixed").Update(mouseMove(point.X, point.Y))
	hover := frameOf(hovered.(tui.Model))
	if !strings.Contains(hover, "#185 Keep service identity explicit") {
		t.Fatalf("hover inspector:\n%s", hover)
	}
	if !strings.Contains(hover, "ready to merge") {
		t.Fatalf("hover headline:\n%s", hover)
	}
}

func TestCompactCardAndHomeEnd(t *testing.T) {
	size := tui.TerminalSize{Width: 40, Height: 20}
	m := applyKey(applyKey(makeUI(size, "mixed"), key("down")), key("end"))
	if !strings.Contains(frameOf(m), "#213 Prepare token migration") {
		t.Fatalf("end of second stack:\n%s", frameOf(m))
	}
	m = applyKey(m, key("home"))
	frame := frameOf(m)
	if !strings.Contains(frame, "#211 Add token catalogue") {
		t.Fatalf("home:\n%s", frame)
	}
	if !strings.Contains(frame, "click checkout · o open") {
		t.Fatalf("compact card hint:\n%s", frame)
	}
	if !strings.Contains(frame, "↑↓ stack · ←→ layer · / find · ? help") {
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
	if tui.GetRowLayout(80, len(first.PRs)).BallsWidth != len(first.PRs)*2 {
		t.Fatal("each layer should own two cells")
	}
}

func TestLocalFilter(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "mixed"), key("/"))
	for _, r := range "composer" {
		m = applyKey(m, key(string(r)))
	}
	frame := frameOf(m)
	if !strings.Contains(frame, "Add ontology tokens to email") {
		t.Fatalf("filtered stack missing:\n%s", frame)
	}
	if strings.Contains(frame, "Simplify authentication boundaries") {
		t.Fatalf("unfiltered stack still visible:\n%s", frame)
	}
}

func TestFixtureCacheAndEmptyStates(t *testing.T) {
	stale := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "draft"))
	if !strings.Contains(stale, "fixture cache · stale (simulated)") {
		t.Fatalf("stale:\n%s", stale)
	}
	empty := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "all-merged"))
	if !strings.Contains(empty, "No open stacks in this fixture repository.") {
		t.Fatalf("empty:\n%s", empty)
	}
	errFrame := frameOf(makeUI(tui.TerminalSize{Width: 80, Height: 24}, "ci-failing"))
	if !strings.Contains(errFrame, "Refresh failed in this fixture. No cached stacks are available.") {
		t.Fatalf("error empty:\n%s", errFrame)
	}
	if !strings.Contains(errFrame, "fixture refresh failed · no cached stacks") {
		t.Fatalf("error status:\n%s", errFrame)
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
	if !strings.Contains(frame, "◉") {
		t.Fatalf("focused layer:\n%s", frame)
	}
	if !strings.Contains(frame, "? help · q quit") {
		t.Fatalf("80-col footer:\n%s", frame)
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
		if !strings.Contains(frame, "STACKS UI LAB") {
			t.Fatalf("%dx%d missing title:\n%s", size.Width, size.Height, frame)
		}
		assertFits(t, frame, size.Width)
		for _, anchor := range []struct{ X, Y int }{
			{0, 0},
			{size.Width - 1, 1},
			{size.Width - 1, size.Height - 3},
			{1, size.Height - 3},
		} {
			placement := tui.ClampCardPlacement(size, anchor)
			if placement.Left < 1 || placement.Top < 1 {
				t.Fatalf("placement underflow at %+v: %+v", anchor, placement)
			}
			if placement.Left+placement.Width > size.Width-1 {
				t.Fatalf("placement overflows right at %+v: %+v", anchor, placement)
			}
			if placement.Top+placement.Height > size.Height-2 {
				t.Fatalf("placement overflows bottom at %+v: %+v", anchor, placement)
			}
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
	if !strings.Contains(frameOf(m), "Opened #184 · fixture simulation") {
		t.Fatalf("open:\n%s", frameOf(m))
	}
	m = applyKey(m, key("r"))
	if !strings.Contains(frameOf(m), "Fixture data refreshed · no network") {
		t.Fatalf("refresh:\n%s", frameOf(m))
	}
}

func TestStoryCycleAndWideInspector(t *testing.T) {
	m := applyKey(makeUI(tui.TerminalSize{Width: 120, Height: 30}, "mixed"), key("]"))
	frame := frameOf(m)
	if !strings.Contains(frame, "all ready") && !strings.Contains(frame, "Story: all ready") {
		t.Fatalf("next story:\n%s", frame)
	}
	wide := frameOf(makeUI(tui.TerminalSize{Width: 120, Height: 30}, "mixed"))
	if !strings.Contains(wide, "#184 Split auth scope from session checks") {
		t.Fatalf("wide inspector should sit beside the list:\n%s", wide)
	}
}
