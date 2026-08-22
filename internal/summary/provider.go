package summary

import (
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// Provider is --provider (example: codex@luna.medium). The summarizer choice
// lives here. Fetch does not depend on it.
type Provider struct {
	Raw   string
	Name  string
	Model string
}

func ParseProvider(raw string) Provider {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Provider{}
	}
	name, model, ok := strings.Cut(raw, "@")
	if !ok {
		return Provider{Raw: raw, Name: raw}
	}
	return Provider{Raw: raw, Name: strings.TrimSpace(name), Model: strings.TrimSpace(model)}
}

// Local is the current summarizer: PreferDescription, then FromLayers.
func Local() Summarizer {
	return PreferDescription{Fallback: FromLayers{}}
}

// Chosen is the Summarizer picked for a provider. Tonight Inner is always
// Local(). No network, no chat API.
type Chosen struct {
	Provider Provider
	Inner    Summarizer
}

func (c Chosen) Summarize(stack domain.Stack) string {
	inner := c.Inner
	if inner == nil {
		inner = Local()
	}
	return inner.Summarize(stack)
}

// Choose threads --provider into the summarizer. Until a network summarizer
// exists, every provider (including empty) gets Local().
func Choose(p Provider) Chosen {
	return Chosen{Provider: p, Inner: Local()}
}

// Apply writes one-line summaries onto stacks using s. Nil s uses Local().
func Apply(stacks []domain.Stack, s Summarizer) []domain.Stack {
	if s == nil {
		s = Local()
	}
	for i := range stacks {
		stacks[i].Summary = s.Summarize(stacks[i])
		if strings.TrimSpace(stacks[i].Description) == "" {
			stacks[i].Description = stacks[i].Summary
		}
	}
	return stacks
}
