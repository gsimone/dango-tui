package data

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// FileDump is an authored stack file. dango.json / dango.yml are provider
// config and are not this format.
type FileDump struct {
	Repo   string      `json:"repo"`
	Stacks []FileStack `json:"stacks"`
}

type FileStack struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	BaseRef     string   `json:"base"`
	PRs         []FilePR `json:"prs"`
}

type FilePR struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Branch       string `json:"branch"`
	Author       string `json:"author"`
	State        string `json:"state"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	ChangedFiles int    `json:"changedFiles"`
}

// IsStackFile reports whether --repo names a JSON stack dump, not owner/name.
func IsStackFile(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	return strings.HasSuffix(strings.ToLower(raw), ".json")
}

// LoadStacks reads an authored stack dump. Missing or wrong-shaped files error;
// this does not invent stacks and does not read dango.json provider config.
func LoadStacks(path string) (string, []domain.Stack, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil, fmt.Errorf("pass --repo owner/name or a JSON stack file")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", path, err)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return "", nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	if _, hasStacks := probe["stacks"]; !hasStacks {
		if _, hasProvider := probe["provider"]; hasProvider {
			return "", nil, fmt.Errorf("%s is provider config, not a stack dump", filepath.Base(path))
		}
		return "", nil, fmt.Errorf("%s: missing stacks", filepath.Base(path))
	}
	var dump FileDump
	if err := json.Unmarshal(raw, &dump); err != nil {
		return "", nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	stacks := make([]domain.Stack, 0, len(dump.Stacks))
	for i, fs := range dump.Stacks {
		stacks = append(stacks, fs.toDomain(i+1))
	}
	repo := strings.TrimSpace(dump.Repo)
	if repo == "" {
		repo = filepath.Base(path)
	}
	return repo, stacks, nil
}

func (s FileStack) toDomain(n int) domain.Stack {
	prs := make([]domain.PullRequest, 0, len(s.PRs))
	for _, fp := range s.PRs {
		prs = append(prs, fp.toDomain())
	}
	name := strings.TrimSpace(s.Name)
	if name == "" {
		name = "stack " + itoa(n)
	}
	id := strings.TrimSpace(s.ID)
	if id == "" {
		id = "file-stack-" + itoa(n)
	}
	base := strings.TrimSpace(s.BaseRef)
	if base == "" {
		base = "main"
	}
	return domain.Stack{
		ID:          id,
		Number:      n,
		Name:        name,
		BaseRef:     base,
		PRs:         prs,
		Description: strings.TrimSpace(s.Description),
	}
}

func (p FilePR) toDomain() domain.PullRequest {
	state := parseFileState(p.State)
	in := PullRequestFixtureInput{
		State:  state,
		Number: p.Number,
		Title:  p.Title,
		URL:    p.URL,
		Branch: p.Branch,
		Author: p.Author,
	}
	if p.Additions != 0 {
		in.Additions = intPtr(p.Additions)
	}
	if p.Deletions != 0 {
		in.Deletions = intPtr(p.Deletions)
	}
	if p.ChangedFiles != 0 {
		in.ChangedFiles = intPtr(p.ChangedFiles)
	}
	return PRFixture(in)
}

func parseFileState(raw string) domain.PrDisplayState {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "merged":
		return domain.StateMerged
	case "draft":
		return domain.StateDraft
	case "ci-failure", "ci failing", "ci_failure":
		return domain.StateCIFailure
	case "review-blocked", "review / merge blocked", "blocked":
		return domain.StateReviewBlocked
	case "queued", "merge queued":
		return domain.StateQueued
	case "ready", "ready to merge":
		return domain.StateReady
	default:
		return domain.StateOpen
	}
}
