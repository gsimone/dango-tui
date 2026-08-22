package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/cli"
	"github.com/gsimone/dango-tui/internal/tui"
)

func main() {
	args, err := cli.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Fprint(os.Stderr, helpText())
		os.Exit(0)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "dango: %v\n", err)
		os.Exit(2)
	}

	model := tui.New(tui.Options{
		StoryID:  args.Story,
		Repo:     args.Repo,
		Provider: args.Provider,
	})

	if args.Frame != "" {
		width, height, err := parseFrame(args.Frame)
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

func helpText() string {
	return "Usage: dango --repo owner/name [--provider name@model]\n" +
		"       dango [-story mixed]\n\n" +
		"Live fetch requires --repo owner/name. No baked-in repo.\n" +
		"--provider is optional (summarizer, e.g. codex@luna.medium) and does not block fetch.\n"
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
