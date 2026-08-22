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

// Catalog is the stub provider/model list. The empty row is off: no generated
// title. Other rows are the same Provider type as --provider.
func Catalog() []Provider {
	return []Provider{
		{},
		ParseProvider("local"),
		ParseProvider("codex@luna.medium"),
		ParseProvider("codex@luna.high"),
	}
}

func (p Provider) Empty() bool {
	return p.Raw == "" && p.Name == ""
}

func (p Provider) Label() string {
	if p.Empty() {
		return "none"
	}
	if p.Raw != "" {
		return p.Raw
	}
	if p.Model != "" {
		return p.Name + "@" + p.Model
	}
	return p.Name
}

func SameProvider(a, b Provider) bool {
	if a.Empty() || b.Empty() {
		return a.Empty() && b.Empty()
	}
	if a.Raw != "" && b.Raw != "" {
		return a.Raw == b.Raw
	}
	return a.Name == b.Name && a.Model == b.Model
}

// Local is the current summarizer: PreferDescription, then FromLayers.
func Local() Summarizer {
	return PreferDescription{Fallback: FromLayers{}}
}

// none writes no stack title. Missing --provider must not invent one.
type none struct{}

func (none) Summarize(domain.Stack) string { return "" }

// Chosen is the Summarizer picked for a provider. --provider is only for the
// stack title. No provider → none. A provider with no network summarizer yet
// uses Local(). Fetch does not wait on this.
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
func Apply(stacks []domain.Stack, s Summarizer) []domain.Stack {
	if s == nil {
		s = none{}
	}
	for i := range stacks {
		title := strings.TrimSpace(s.Summarize(stacks[i]))
		stacks[i].Summary = title
		if title != "" {
			stacks[i].Name = title
		}
	}
	return stacks
}
