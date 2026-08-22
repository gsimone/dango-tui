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
	RootPaddingX = 1
	HeaderRows   = 2
	AirRows      = 2
	ListStartY   = HeaderRows + AirRows
)

func GetRowLayout(width int, prCount int) RowLayout {
	return rowLayout(width, prCount, width <= 50)
}

func GetListRowLayout(listWidth, termWidth, prCount int) RowLayout {
	return rowLayout(listWidth, prCount, termWidth <= 50)
}

func rowLayout(width int, prCount int, compact bool) RowLayout {
	contentWidth := max(1, width-RootPaddingX*2)
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

func InspectorColumnWidth(width int) int {
	if width <= 50 {
		return max(14, min(16, max(1, width-2)/2))
	}
	if width >= 100 {
		return min(48, max(40, width*38/100))
	}
	return max(36, min(40, width-42))
}

func ListTerminalWidth(termWidth int, inspectorWidth int) int {
	if inspectorWidth <= 0 {
		inspectorWidth = InspectorColumnWidth(termWidth)
	}
	left := termWidth - 1 - inspectorWidth
	if left < 1 {
		left = 1
	}
	return max(12, left-1)
}

func GetBallPoint(size TerminalSize, stackIndex, prIndex, prCount int) struct{ X, Y int } {
	listWidth := ListTerminalWidth(size.Width, InspectorColumnWidth(size.Width))
	layout := GetListRowLayout(listWidth, size.Width, prCount)
	return struct{ X, Y int }{
		X: RootPaddingX + layout.NameWidth + 1 + prIndex*2,
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
	left := size.Width - 1 - width
	if left < 1 {
		left = 1
	}
	return CardPlacement{
		Left:    left,
		Top:     ListStartY,
		Width:   width,
		Height:  max(3, size.Height-2-ListStartY),
		Compact: size.Width <= 50,
	}
}

func ClampCardPlacement(size TerminalSize, _ struct{ X, Y int }) CardPlacement {
	place := GetInspectorSize(size)
	if place.Left < 1 {
		place.Left = 1
	}
	if place.Left+place.Width > size.Width {
		place.Width = max(1, size.Width-place.Left)
	}
	if place.Top < 1 {
		place.Top = 1
	}
	if place.Top+place.Height > size.Height-2 {
		place.Height = max(3, size.Height-2-place.Top)
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
