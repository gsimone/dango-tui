package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/cli"
	"github.com/gsimone/dango-tui/internal/tui"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// startTUI opens the alt-screen app. Tests replace this so --doctor
// cannot silently construct a program.
var startTUI = func(m tui.Model) error {
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run()
	return err
}

func run(argv []string, stdout, stderr io.Writer) int {
	args, err := cli.Parse(argv)
	if err == flag.ErrHelp {
		fmt.Fprint(stderr, helpText())
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "dango: %v\n", err)
		return 2
	}

	if args.Doctor {
		if err := cli.RunDoctor(stdout); err != nil {
			return 2
		}
		return 0
	}

	args, err = cli.ResolveLaunch(args)
	if err != nil {
		fmt.Fprintf(stderr, "dango: %v\n", err)
		return 2
	}

	model := tui.New(tui.Options{
		StoryID:     args.Story,
		Repo:        args.Repo,
		Provider:    args.Provider,
		Describe:    args.Describe,
		DescribeDir: args.DescribeDir,
	})

	if args.Frame != "" {
		width, height, err := parseFrame(args.Frame)
		if err != nil {
			fmt.Fprintf(stderr, "dango: %v\n", err)
			return 2
		}
		fmt.Fprintln(stdout, stripTrailing(model.RenderFrame(width, height)))
		return 0
	}

	if err := startTUI(model); err != nil {
		fmt.Fprintf(stderr, "dango: %v\n", err)
		return 1
	}
	return 0
}

func helpText() string {
	return cli.Usage()
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
