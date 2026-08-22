package tui

type TerminalSize struct {
	Width  int
	Height int
}

type RowLayout struct {
	Compact          bool
	NameWidth        int
	BallsWidth       int
	DescriptionWidth int
}

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

func rowLayout(contentWidth int, prCount int, compact bool) RowLayout {
	if contentWidth < 1 {
		contentWidth = 1
	}
	ballsWidth := prCount * 2
	desiredName := 22
	if compact {
		desiredName = 14
	}
	gap := 1
	nameWidth := max(8, min(desiredName, contentWidth-ballsWidth-gap))
	return RowLayout{
		Compact:          compact,
		NameWidth:        nameWidth,
		BallsWidth:       ballsWidth,
		DescriptionWidth: max(0, contentWidth-nameWidth-ballsWidth-gap),
	}
}

func innerWidth(termWidth int) int {
	return max(1, termWidth-PadX*2)
}

func InspectorColumnWidth(width int) int {
	inner := innerWidth(width)
	if width <= 50 {
		return 14
	}
	if width >= 100 {
		return min(48, max(40, inner*38/100))
	}
	return max(32, min(38, inner-40))
}

func ListPaneWidth(termWidth int) int {
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
	return struct{ X, Y int }{
		X: PadX + layout.NameWidth + 1 + prIndex*2,
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
