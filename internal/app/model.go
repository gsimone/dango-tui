package app

import (
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

type Selection struct {
	StackIndex int
	PRIndex    int
}

type State struct {
	Selection   Selection
	Query       string
	Searching   bool
	CardVisible bool
	Feedback    string
}

func InitialState() State {
	return State{
		Selection:   Selection{StackIndex: 0, PRIndex: 0},
		Query:       "",
		Searching:   false,
		CardVisible: true,
		Feedback:    "",
	}
}

func FilterStacks(stacks []domain.Stack, query string) []domain.Stack {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return stacks
	}
	out := make([]domain.Stack, 0, len(stacks))
	for _, stack := range stacks {
		if strings.Contains(strings.ToLower(stack.Name), needle) {
			out = append(out, stack)
			continue
		}
		matched := false
		for _, pr := range stack.PRs {
			if strings.Contains(strings.ToLower(pr.Title), needle) {
				matched = true
				break
			}
		}
		if matched {
			out = append(out, stack)
		}
	}
	return out
}

func ClampSelection(selection Selection, stacks []domain.Stack) Selection {
	if len(stacks) == 0 {
		return Selection{StackIndex: 0, PRIndex: 0}
	}
	stackIndex := selection.StackIndex
	if stackIndex < 0 {
		stackIndex = 0
	}
	if stackIndex > len(stacks)-1 {
		stackIndex = len(stacks) - 1
	}
	prs := stacks[stackIndex].PRs
	prIndex := selection.PRIndex
	if prIndex < 0 {
		prIndex = 0
	}
	maxPR := len(prs) - 1
	if maxPR < 0 {
		maxPR = 0
	}
	if prIndex > maxPR {
		prIndex = maxPR
	}
	return Selection{StackIndex: stackIndex, PRIndex: prIndex}
}

type Direction string

const (
	DirUp    Direction = "up"
	DirDown  Direction = "down"
	DirLeft  Direction = "left"
	DirRight Direction = "right"
	DirHome  Direction = "home"
	DirEnd   Direction = "end"
)

func MoveSelection(selection Selection, stacks []domain.Stack, direction Direction) Selection {
	current := ClampSelection(selection, stacks)
	switch direction {
	case DirLeft:
		return ClampSelection(Selection{StackIndex: current.StackIndex, PRIndex: current.PRIndex - 1}, stacks)
	case DirRight:
		return ClampSelection(Selection{StackIndex: current.StackIndex, PRIndex: current.PRIndex + 1}, stacks)
	case DirHome:
		return ClampSelection(Selection{StackIndex: 0, PRIndex: current.PRIndex}, stacks)
	case DirEnd:
		last := 0
		if len(stacks) > 0 {
			last = len(stacks) - 1
		}
		return ClampSelection(Selection{StackIndex: last, PRIndex: current.PRIndex}, stacks)
	default:
		delta := 1
		if direction == DirUp {
			delta = -1
		}
		return ClampSelection(Selection{StackIndex: current.StackIndex + delta, PRIndex: current.PRIndex}, stacks)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
