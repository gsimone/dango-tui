package summary

import (
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
// gh name. Fetch and first paint do not call this.
func Run(job Job) Result {
	id := job.ID
	if id == "" {
		id = job.Stack.ID
	}
	if job.Provider.Empty() {
		return Result{ID: id}
	}
	title := strings.TrimSpace(Choose(job.Provider).Summarize(job.Stack))
	desc := Describe(job.Stack)
	return Result{ID: id, Title: title, Description: desc}
}

// Describe writes inspector copy. A real non-stub Description wins;
// otherwise one clause from the layer titles. One line. Never a title.
func Describe(stack domain.Stack) string {
	desc := strings.TrimSpace(stack.Description)
	if desc != "" && desc != "A deterministic fixture stack" {
		return desc
	}
	return strings.TrimSpace(FromLayers{}.Summarize(stack))
}
