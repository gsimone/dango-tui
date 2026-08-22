package summary

import "github.com/gsimone/dango-tui/internal/domain"

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

// Run is the provider hook. Tonight it no-ops: there is no Codex client.
// Replace this body later; do not call it from New or the first View.
func Run(job Job) Result {
	id := job.ID
	if id == "" {
		id = job.Stack.ID
	}
	return Result{ID: id}
}
