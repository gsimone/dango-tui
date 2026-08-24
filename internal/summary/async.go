package summary

import (
	"strings"
	"unicode/utf8"

	"github.com/gsimone/dango-tui/internal/domain"
)

// Job is one stack-title request. Fetch does not wait on it.
type Job struct {
	Provider Provider
	Describe string
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

// Run fills the inspector description after first paint. The product
// sentence comes from the configured describe script (dango.json
// `describe` / --describe). Local Describe() is the fallback when the
// script is missing, non-zero, times out, or returns mush. A title is
// written only when a provider is set, so the list keeps the gh /
// short name. Fetch and first paint do not call this.
func Run(job Job) Result {
	id := job.ID
	if id == "" {
		id = job.Stack.ID
	}
	res := Result{ID: id}
	if desc, err := describeScript(job); err == nil && strings.TrimSpace(desc) != "" {
		res.Description = strings.TrimSpace(desc)
	} else {
		res.Description = Describe(job.Stack)
	}
	if !job.Provider.Empty() {
		res.Title = Title(job.Stack)
	}
	return res
}

func stripTicket(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSpace(stripIssuePrefix(s))
	s = strings.TrimSpace(stripHashPrefix(s))
	return s
}

func stripIssuePrefix(s string) string {
	i := 0
	if i < len(s) && s[i] == '[' {
		i++
	}
	if i >= len(s) || !isASCIILetter(s[i]) {
		return s
	}
	i++
	for i < len(s) && isASCIIAlnum(s[i]) {
		i++
	}
	if i >= len(s) || s[i] != '-' {
		return s
	}
	i++
	if i >= len(s) || s[i] < '0' || s[i] > '9' {
		return s
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i < len(s) && s[i] == ']' {
		i++
	}
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) || !isTicketSep(s[i:]) {
		return s
	}
	_, w := utf8.DecodeRuneInString(s[i:])
	i += w
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func stripHashPrefix(s string) string {
	if !strings.HasPrefix(s, "#") {
		return s
	}
	i := 1
	if i >= len(s) || s[i] < '0' || s[i] > '9' {
		return s
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if i >= len(s) || !isTicketSep(s[i:]) {
		return s
	}
	_, w := utf8.DecodeRuneInString(s[i:])
	i += w
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return s[i:]
}

func isTicketSep(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return r == ':' || r == '-' || r == '\u2013' || r == '\u2014'
}

func isASCIILetter(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func isASCIIAlnum(c byte) bool {
	return isASCIILetter(c) || c >= '0' && c <= '9'
}

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

// Describe writes a two-line meta clause from the layers when that
// clause is not the raw gh title. It never pastes pr.Body, never wraps
// with an invented "Covers …" prefix, and leaves the slot empty when
// it cannot write a distinct sentence.
func Describe(stack domain.Stack) string {
	gh := ghName(stack)
	if clause := clauseFromLayers(stack); clause != "" && !sameFold(clause, gh) {
		return clause
	}
	if stripped := stripTicket(gh); stripped != "" && !sameFold(stripped, gh) {
		return stripped
	}
	return ""
}
