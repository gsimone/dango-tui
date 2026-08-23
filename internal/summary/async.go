package summary

import (
	"regexp"
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// Job is one stack-title request. Fetch does not wait on it.
type Job struct {
	Provider Provider
	Stack    domain.Stack
	ID       string
}

// Result lands after first paint. Empty Title/Description means no fill.
type Result struct {
	ID          string
	Title       string
	Description string
}

// Func is one provider hook. Tests inject a fake; production uses Run.
type Func func(Job) Result

// Run generates a short stack title and an inspector description when the
// job has a provider. Missing provider returns empty fills and keeps the
// gh name. Fetch and first paint do not call this. local/demo write a
// distinct clause — they never echo the raw gh title.
func Run(job Job) Result {
	id := job.ID
	if id == "" {
		id = job.Stack.ID
	}
	if job.Provider.Empty() {
		return Result{ID: id}
	}
	return Result{ID: id, Title: Title(job.Stack), Description: Describe(job.Stack)}
}

var (
	ticketPrefix = regexp.MustCompile(`(?i)^(?:\[)?[A-Z][A-Z0-9]*-\d+(?:\])?\s*[:\-\x{2013}\x{2014}]\s*`)
	hashPrefix   = regexp.MustCompile(`^#\d+\s*[:\-\x{2013}\x{2014}]\s*`)
)

func ghName(stack domain.Stack) string {
	for _, pr := range stack.PRs {
		if t := strings.TrimSpace(pr.Title); t != "" {
			return t
		}
		if strings.TrimSpace(pr.Branch) != "" {
			return strings.TrimSpace(pr.Branch)
		}
	}
	return strings.TrimSpace(stack.Name)
}

func sameFold(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func stripTicket(s string) string {
	s = strings.TrimSpace(s)
	s = ticketPrefix.ReplaceAllString(s, "")
	s = hashPrefix.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func layerTitles(stack domain.Stack) []string {
	var out []string
	for _, pr := range stack.PRs {
		t := stripTicket(strings.TrimSpace(pr.Title))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func clauseFromLayers(stack domain.Stack) string {
	titles := layerTitles(stack)
	if len(titles) == 0 {
		return ""
	}
	return strings.TrimSpace(joinTitles(titles))
}

func distinct(s, gh, title string) bool {
	s = strings.TrimSpace(s)
	return s != "" && !sameFold(s, gh) && !sameFold(s, title)
}

// Title is a short generated list name. Empty means keep the gh name.
// It never pastes the raw gh title back.
func Title(stack domain.Stack) string {
	gh := ghName(stack)
	if clause := clauseFromLayers(stack); distinct(clause, gh, "") {
		return clause
	}
	if stripped := stripTicket(gh); distinct(stripped, gh, "") {
		return stripped
	}
	return ""
}

// Describe writes one invented inspector sentence. It never pastes
// pr.Body, HTML comments, CURSOR_AGENT markers, or markdown links.
func Describe(stack domain.Stack) string {
	gh := ghName(stack)
	title := Title(stack)
	if clause := clauseFromLayers(stack); clause != "" {
		desc := "Covers " + clause + "."
		if distinct(desc, gh, title) {
			return desc
		}
	}
	if gh != "" {
		return "Covers this stack."
	}
	return ""
}
