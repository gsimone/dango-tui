package tui

type TerminalSize struct {
	Width  int
	Height int
}

type RowLayout struct {
	Compact          bool
	NameWidth        int
	BallsWidth       int
	StatusWidth      int
	DescriptionWidth int
}

const (
	BallColW   = 10
	StatusColW = 9
)

const (
	PadX       = 2
	PadTop     = 1
	HeaderRows = 2
	AirRows    = 2
	ListStartY = PadTop + HeaderRows + AirRows
)

func GetRowLayout(width int, prCount int) RowLayout {
	return rowLayout(max(1, width-PadX*2), prCount, width <= 50)
}

func GetListRowLayout(listWidth, termWidth, prCount int) RowLayout {
	return rowLayout(listWidth, prCount, termWidth <= 50)
}

func lockedNameWidth(contentWidth int, compact bool) int {
	desired := 18
	if compact {
		desired = 14
	}
	return max(8, min(desired, max(8, contentWidth-6)))
}

func rowLayout(contentWidth int, prCount int, compact bool) RowLayout {
	if contentWidth < 1 {
		contentWidth = 1
	}
	balls := BallColW
	nameWidth := lockedNameWidth(contentWidth, compact)
	need := nameWidth + 1 + balls + 1 + StatusColW
	if need > contentWidth {
		nameWidth = max(8, contentWidth-balls-StatusColW-2)
	}
	return RowLayout{
		Compact:          compact,
		NameWidth:        nameWidth,
		BallsWidth:       balls,
		StatusWidth:      StatusColW,
		DescriptionWidth: StatusColW,
	}
}

func innerWidth(termWidth int) int {
	return max(1, termWidth-PadX*2)
}

func StackedInspector(termWidth int) bool { return termWidth < 100 }

func InspectorColumnWidth(width int) int {
	inner := innerWidth(width)
	if StackedInspector(width) {
		return inner
	}
	return min(48, max(40, inner*38/100))
}

func ListPaneWidth(termWidth int) int {
	if StackedInspector(termWidth) {
		return innerWidth(termWidth)
	}
	insp := InspectorColumnWidth(termWidth)
	return max(12, innerWidth(termWidth)-1-insp)
}

func ListTerminalWidth(termWidth int, _ int) int {
	return ListPaneWidth(termWidth)
}

func RuleX(termWidth int) int {
	return PadX + ListPaneWidth(termWidth)
}

func InspectorLeft(termWidth int) int {
	return RuleX(termWidth) + 1
}

func GetBallPoint(size TerminalSize, stackIndex, prIndex, prCount int) struct{ X, Y int } {
	listWidth := ListPaneWidth(size.Width)
	layout := GetListRowLayout(listWidth, size.Width, prCount)
	x := ballCellX(PadX+layout.NameWidth+1, prCount, prIndex)
	return struct{ X, Y int }{
		X: x,
		Y: ListStartY + stackIndex,
	}
}

type CardPlacement struct {
	Left    int
	Top     int
	Width   int
	Height  int
	Compact bool
}

func GetInspectorSize(size TerminalSize) CardPlacement {
	if StackedInspector(size.Width) {
		width := innerWidth(size.Width)
		top := ListStartY + 1
		return CardPlacement{
			Left:    PadX,
			Top:     top,
			Width:   width,
			Height:  max(3, size.Height-1-top),
			Compact: size.Width <= 50,
		}
	}
	width := InspectorColumnWidth(size.Width)
	left := InspectorLeft(size.Width)
	rightLimit := size.Width - PadX
	if left+width > rightLimit {
		width = max(1, rightLimit-left)
	}
	return CardPlacement{
		Left:    left,
		Top:     ListStartY,
		Width:   width,
		Height:  max(3, size.Height-1-ListStartY),
		Compact: size.Width <= 50,
	}
}

func ClampCardPlacement(size TerminalSize, _ struct{ X, Y int }) CardPlacement {
	place := GetInspectorSize(size)
	if place.Left < PadX {
		place.Left = PadX
	}
	if place.Left+place.Width > size.Width-PadX {
		place.Width = max(1, size.Width-PadX-place.Left)
	}
	if place.Top < 1 {
		place.Top = 1
	}
	if place.Top+place.Height > size.Height-1 {
		place.Height = max(3, size.Height-1-place.Top)
	}
	return place
}

func IsWide(width int) bool    { return width >= 100 }
func IsCompact(width int) bool { return width <= 50 }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
