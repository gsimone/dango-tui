package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/tui"
)

func main() {
	frame := flag.String("frame", "", "print a fixture frame (WxH, e.g. 80x24) and exit")
	flag.Parse()

	model := tui.New(tui.Options{})

	if *frame != "" {
		width, height, err := parseFrame(*frame)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dango: %v\n", err)
			os.Exit(2)
		}
		fmt.Println(stripTrailing(model.RenderFrame(width, height)))
		return
	}

	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "dango: %v\n", err)
		os.Exit(1)
	}
}

func parseFrame(spec string) (int, int, error) {
	parts := strings.Split(strings.ToLower(spec), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("frame must look like 80x24")
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil || width < 1 {
		return 0, 0, fmt.Errorf("frame width must be a positive integer")
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil || height < 1 {
		return 0, 0, fmt.Errorf("frame height must be a positive integer")
	}
	return width, height, nil
}

func stripTrailing(s string) string {
	return strings.TrimRight(s, "\n")
}
