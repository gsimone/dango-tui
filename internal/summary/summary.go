package summary

import (
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// Summarizer turns a stack into one short clause. No network.
type Summarizer interface {
	Summarize(stack domain.Stack) string
}

// FromDescription passes through an existing description.
type FromDescription struct{}

func (FromDescription) Summarize(stack domain.Stack) string {
	return strings.TrimSpace(stack.Description)
}

// FromLayers writes one short clause from the stack's PR titles.
type FromLayers struct{}

func (FromLayers) Summarize(stack domain.Stack) string {
	if len(stack.PRs) == 0 {
		if stack.Name == "" {
			return "no layers"
		}
		return "no layers in " + stack.Name
	}

	titles := make([]string, 0, len(stack.PRs))
	for _, pr := range stack.PRs {
		title := strings.TrimSpace(pr.Title)
		if title != "" {
			titles = append(titles, title)
		}
	}
	if len(titles) == 0 {
		return "no titled layers in " + stack.Name
	}
	return joinTitles(titles)
}

// PreferDescription uses a non-stub Description, otherwise the fallback.
type PreferDescription struct {
	Fallback Summarizer
}

func (p PreferDescription) Summarize(stack domain.Stack) string {
	desc := strings.TrimSpace(stack.Description)
	if desc != "" && desc != "A deterministic fixture stack" {
		return desc
	}
	if p.Fallback != nil {
		return p.Fallback.Summarize(stack)
	}
	return ""
}

func joinTitles(titles []string) string {
	lowered := make([]string, len(titles))
	for i, title := range titles {
		lowered[i] = lowerStart(title)
	}
	switch len(lowered) {
	case 1:
		return lowered[0]
	case 2:
		return lowered[0] + " and " + lowered[1]
	default:
		return strings.Join(lowered[:len(lowered)-1], ", ") + ", and " + lowered[len(lowered)-1]
	}
}

func lowerStart(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	return strings.ToLower(string(runes[0])) + string(runes[1:])
}
