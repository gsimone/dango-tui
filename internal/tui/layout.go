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
	ListStartY   = 3
)

func GetRowLayout(width int, prCount int) RowLayout {
	contentWidth := max(1, width-RootPaddingX*2)
	ballsWidth := prCount * 2
	compact := width <= 50
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

func GetBallPoint(size TerminalSize, stackIndex, prIndex, prCount int) struct{ X, Y int } {
	layout := GetRowLayout(size.Width, prCount)
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

// GetInspectorSize is a stable inspector pane: alongside the list when there
// is room, below it otherwise.
func GetInspectorSize(size TerminalSize) CardPlacement {
	compact := size.Width <= 50
	width := max(16, size.Width-2)
	if !compact && size.Width >= 100 {
		width = min(48, max(40, size.Width*38/100))
	}
	height := 9
	if compact {
		height = 8
	}
	return CardPlacement{Width: width, Height: height, Compact: compact}
}

// ClampCardPlacement keeps a sibling overlay inside the usable terminal,
// clear of status/footer.
func ClampCardPlacement(size TerminalSize, anchor struct{ X, Y int }) CardPlacement {
	compact := size.Width <= 50
	maxCard := 56
	if compact {
		maxCard = 38
	}
	cardWidth := max(16, min(maxCard, size.Width-2))
	cardHeight := 9
	if compact {
		cardHeight = 8
	}
	maxLeft := max(1, size.Width-cardWidth-1)
	usableBottom := max(1, size.Height-2)
	below := anchor.Y + 1
	top := below
	if below+cardHeight > usableBottom {
		top = max(1, anchor.Y-cardHeight)
	}
	place := CardPlacement{
		Left:    min(max(1, anchor.X+2), maxLeft),
		Top:     top,
		Width:   cardWidth,
		Height:  min(cardHeight, max(3, usableBottom-top)),
		Compact: compact,
	}
	if place.Left+place.Width > size.Width-1 {
		place.Width = max(16, size.Width-1-place.Left)
	}
	if place.Top+place.Height > usableBottom {
		place.Height = max(3, usableBottom-place.Top)
	}
	if place.Top < 1 {
		place.Top = 1
	}
	return place
}

func IsWide(width int) bool    { return width >= 100 }
func IsCompact(width int) bool { return width <= 50 }

func ListTerminalWidth(termWidth int, _ int) int {
	// The postcard overlays the field; do not reserve a chrome column.
	return termWidth
}

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
