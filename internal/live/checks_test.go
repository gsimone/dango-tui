package live

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestChecksArgsStayOffTheFirstList(t *testing.T) {
	args := ChecksArgs("archetype-labs/app", 12)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "pr checks 12") || !strings.Contains(joined, "--json") {
		t.Fatalf("checks argv %v", args)
	}
	if strings.Contains(joined, "statusCheckRollup") || strings.Contains(joined, "pr list") {
		t.Fatalf("follow-up must not be the first list: %v", args)
	}
	first := strings.Join(prListFields, ",")
	if strings.Contains(first, "statusCheckRollup") {
		t.Fatal("first pr list must stay off statusCheckRollup")
	}
}

func TestEnrichCIMapsFailBucket(t *testing.T) {
	run := func(args ...string) ([]byte, error) {
		if args[0] != "pr" || args[1] != "checks" {
			t.Fatalf("only pr checks, got %v", args)
		}
		switch args[2] {
		case "1":
			return []byte(`[{"bucket":"fail","state":"FAILURE"},{"bucket":"pass","state":"SUCCESS"}]`), nil
		case "2":
			return []byte(`[{"bucket":"pass","state":"SUCCESS"}]`), nil
		default:
			return nil, fmt.Errorf("unexpected %v", args)
		}
	}
	stacks := []domain.Stack{{
		ID: "s",
		PRs: []domain.PullRequest{
			{Number: 1, Title: "base"},
			{Number: 2, Title: "head"},
		},
	}}
	got := enrichCIWith(run, "archetype-labs/app", stacks)
	if got[0].PRs[0].CI.State != domain.CIFailure || got[0].PRs[0].CI.Failed != 1 {
		t.Fatalf("fail bucket: %+v", got[0].PRs[0].CI)
	}
	if got[0].PRs[1].CI.State != domain.CISuccess {
		t.Fatalf("pass bucket: %+v", got[0].PRs[1].CI)
	}
	if domain.GetDisplayState(got[0].PRs[0]) != domain.StateCIFailure {
		t.Fatal("fail wins the ball")
	}
}

func TestEnrichCIFailureLeavesCIUnknown(t *testing.T) {
	run := func(args ...string) ([]byte, error) {
		return nil, fmt.Errorf("HTTP 502")
	}
	stacks := []domain.Stack{{PRs: []domain.PullRequest{{Number: 9, Title: "x"}, {Number: 10, Title: "y"}}}}
	got := enrichCIWith(run, "archetype-labs/app", stacks)
	if got[0].PRs[0].CI.State == domain.CIFailure || got[0].PRs[0].CI.Failed > 0 {
		t.Fatalf("502 must not invent CI: %+v", got[0].PRs[0].CI)
	}
}

func TestApplyCIMergesByNumber(t *testing.T) {
	dst := []domain.Stack{{PRs: []domain.PullRequest{{Number: 1}, {Number: 2}}}}
	src := []domain.Stack{{PRs: []domain.PullRequest{
		{Number: 2, CI: domain.CISummary{State: domain.CIFailure, Failed: 1, Total: 1}},
	}}}
	got := ApplyCI(dst, src)
	if got[0].PRs[1].CI.State != domain.CIFailure {
		t.Fatalf("apply: %+v", got[0].PRs)
	}
}
