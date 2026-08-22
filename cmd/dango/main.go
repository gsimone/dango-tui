package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/tui"
)

func main() {
	stories := flag.Bool("stories", false, "open the fixture UI lab")
	story := flag.String("story", "", "fixture story id")
	frame := flag.String("frame", "", "print a fixture frame (WxH, e.g. 80x24) and exit")
	flag.Parse()

	storyID := *story
	if storyID == "" {
		storyID = os.Getenv("STACKS_STORY")
	}
	mode := tui.ModeStacks
	if *stories || storyID != "" {
		mode = tui.ModeStories
	}
	if storyID != "" && !data.IsFixtureStoryID(storyID) {
		fmt.Fprintf(os.Stderr, "dango: unknown fixture story %q\n", storyID)
		os.Exit(2)
	}

	model := tui.New(tui.Options{
		Mode:    mode,
		StoryID: storyID,
	})

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
