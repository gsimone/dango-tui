package live

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// EnrichCIFunc fills CI after first paint. Tests inject a fake.
type EnrichCIFunc func(repo string, stacks []domain.Stack) []domain.Stack

// ChecksArgs is the follow-up argv for one PR. Not on the first pr list.
func ChecksArgs(repo string, number int) []string {
	if strings.TrimSpace(repo) == "" {
		repo = "archetype-labs/app"
	}
	return []string{"pr", "checks", strconv.Itoa(number), "--repo", repo, "--json", "bucket,state"}
}

// EnrichCI loads CI via `gh pr checks` after the list exists.
// It does not call statusCheckRollup and does not rewrite LastGHArgv.
func EnrichCI(repo string, stacks []domain.Stack) []domain.Stack {
	return enrichCIWith(runGHKeep, repo, stacks)
}

func enrichCIWith(run runner, repo string, stacks []domain.Stack) []domain.Stack {
	if len(stacks) == 0 {
		return stacks
	}
	out := append([]domain.Stack(nil), stacks...)
	for i := range out {
		out[i].PRs = append([]domain.PullRequest(nil), out[i].PRs...)
		for j := range out[i].PRs {
			num := out[i].PRs[j].Number
			if num <= 0 {
				continue
			}
			raw, err := run(ChecksArgs(repo, num)...)
			if err != nil && len(raw) == 0 {
				continue
			}
			var rows []ghCheckRow
			if json.Unmarshal(raw, &rows) != nil {
				continue
			}
			out[i].PRs[j].CI = ciFromChecks(rows)
		}
	}
	return out
}

// ApplyCI copies CI summaries onto matching PR numbers.
func ApplyCI(dst, src []domain.Stack) []domain.Stack {
	byNum := map[int]domain.CISummary{}
	for _, stack := range src {
		for _, pr := range stack.PRs {
			if pr.Number > 0 {
				byNum[pr.Number] = pr.CI
			}
		}
	}
	for i := range dst {
		for j := range dst[i].PRs {
			if ci, ok := byNum[dst[i].PRs[j].Number]; ok {
				dst[i].PRs[j].CI = ci
			}
		}
	}
	return dst
}

type ghCheckRow struct {
	Bucket string `json:"bucket"`
	State  string `json:"state"`
}

func ciFromChecks(rows []ghCheckRow) domain.CISummary {
	total := len(rows)
	failed, pending := 0, 0
	for _, row := range rows {
		token := strings.ToLower(strings.TrimSpace(row.Bucket))
		if token == "" {
			token = strings.ToLower(strings.TrimSpace(row.State))
		}
		switch token {
		case "fail", "failure", "error", "cancelled", "canceled", "timed_out", "action_required":
			failed++
		case "pending", "queued", "expected", "in_progress":
			pending++
		}
	}
	switch {
	case failed > 0:
		return domain.CISummary{State: domain.CIFailure, Failed: failed, Pending: pending, Total: total}
	case pending > 0:
		return domain.CISummary{State: domain.CIPending, Pending: pending, Total: total}
	case total > 0:
		return domain.CISummary{State: domain.CISuccess, Total: total}
	default:
		return domain.CISummary{}
	}
}

// runGHKeep returns stdout even when gh exits 8 (checks pending).
// It does not record LastGHArgv — the splash copies the first pr list.
func runGHKeep(args ...string) ([]byte, error) {
	cmd := ghCommand(args...)
	out, err := cmd.Output()
	if len(out) > 0 {
		return out, nil
	}
	return out, err
}
