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

func (p Provider) Empty() bool {
	return p.Raw == "" && p.Name == ""
}

type titleWriter struct{}

func (titleWriter) Summarize(stack domain.Stack) string { return Title(stack) }

// Local is the current summarizer: PreferDescription, then a short
// generated title that does not echo the gh name.
func Local() Summarizer {
	return PreferDescription{Fallback: titleWriter{}}
}

// none writes no stack title. Missing --provider must not invent one.
type none struct{}

func (none) Summarize(domain.Stack) string { return "" }

// Chosen is the Summarizer picked for a provider. --provider writes the
// stack title. No provider → none (list keeps the gh / short name).
// Run fills the inspector via luna (gpt-5.6), then Describe() on
// failure. A provider with no network title client yet uses Local().
// Fetch does not wait on this.
type Chosen struct {
	Provider Provider
	Inner    Summarizer
}

func (c Chosen) Summarize(stack domain.Stack) string {
	if c.Inner != nil {
		return c.Inner.Summarize(stack)
	}
	if c.Provider.Empty() {
		return ""
	}
	return Local().Summarize(stack)
}

// Choose threads --provider into the stack-title summarizer.
func Choose(p Provider) Chosen {
	if p.Empty() {
		return Chosen{Provider: p, Inner: none{}}
	}
	return Chosen{Provider: p, Inner: Local()}
}

// Apply writes a generated stack title when s returns one. Empty/nil s
// leaves Name alone — it does not invent a local title to fill the row.
// A generated title also fills a missing Description for the inspector.
func Apply(stacks []domain.Stack, s Summarizer) []domain.Stack {
	if s == nil {
		s = none{}
	}
	for i := range stacks {
		title := strings.TrimSpace(s.Summarize(stacks[i]))
		stacks[i].Summary = title
		if title != "" {
			stacks[i].Name = title
			if strings.TrimSpace(stacks[i].Description) == "" || stacks[i].Description == "A deterministic fixture stack" {
				stacks[i].Description = Describe(stacks[i])
			}
		}
	}
	return stacks
}
