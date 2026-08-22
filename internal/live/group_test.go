package live

import (
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestGroupStacksWalksBaseChains(t *testing.T) {
	prs := []RemotePR{
		{Number: 10, Title: "base layer", HeadRefName: "feat/a", BaseRefName: "main"},
		{Number: 11, Title: "mid layer", HeadRefName: "feat/b", BaseRefName: "feat/a"},
		{Number: 12, Title: "head layer", HeadRefName: "feat/c", BaseRefName: "feat/b"},
		{Number: 20, Title: "solo", HeadRefName: "fix/one", BaseRefName: "main"},
	}
	stacks := GroupStacks(prs, "main")
	if len(stacks) != 2 {
		t.Fatalf("got %d stacks", len(stacks))
	}
	var chain, solo []int
	for _, stack := range stacks {
		nums := numbers(stack.PRs)
		if len(nums) == 3 {
			chain = nums
		}
		if len(nums) == 1 {
			solo = nums
		}
	}
	if strings.Join(itoaAll(chain), ",") != "10,11,12" {
		t.Fatalf("chain %v", chain)
	}
	if strings.Join(itoaAll(solo), ",") != "20" {
		t.Fatalf("solo %v", solo)
	}
	if stackKey(stacks[0]) != 20 {
		t.Fatalf("newer head should sort first, got %d", stackKey(stacks[0]))
	}
}

func TestGroupStacksUsesGraphiteBodyList(t *testing.T) {
	body := "Managed by Graphite.\n- #4\n- #5\n- #6\n"
	prs := []RemotePR{
		{Number: 4, Title: "one", HeadRefName: "g/a", BaseRefName: "main", Body: body},
		{Number: 5, Title: "two", HeadRefName: "g/b", BaseRefName: "main", Body: body},
		{Number: 6, Title: "three", HeadRefName: "g/c", BaseRefName: "main", Body: body},
	}
	stacks := GroupStacks(prs, "main")
	if len(stacks) != 1 {
		t.Fatalf("graphite list should be one stack, got %d", len(stacks))
	}
	if got := strings.Join(itoaAll(numbers(stacks[0].PRs)), ","); got != "4,5,6" {
		t.Fatalf("order %s", got)
	}
	if !strings.HasPrefix(stacks[0].ID, "graphite-") {
		t.Fatalf("id %s", stacks[0].ID)
	}
}

func TestGroupStacksUsesGitHubNativeStack(t *testing.T) {
	prs := []RemotePR{
		{Number: 1, Title: "bottom", HeadRefName: "s/a", BaseRefName: "main", StackNumber: 7, StackPosition: 1},
		{Number: 2, Title: "top", HeadRefName: "s/b", BaseRefName: "main", StackNumber: 7, StackPosition: 2},
		{Number: 9, Title: "other", HeadRefName: "solo", BaseRefName: "main"},
	}
	stacks := GroupStacks(prs, "main")
	if len(stacks) != 2 {
		t.Fatalf("got %d", len(stacks))
	}
	var native []int
	for _, stack := range stacks {
		if stack.ID == "gh-stack-7" {
			native = numbers(stack.PRs)
		}
	}
	if strings.Join(itoaAll(native), ",") != "1,2" {
		t.Fatalf("native %v", native)
	}
}

func TestGroupStacksDoesNotInventLeva(t *testing.T) {
	if stacks := GroupStacks(nil, "main"); len(stacks) != 0 {
		t.Fatal("empty repo must stay empty")
	}
}

func numbers(prs []domain.PullRequest) []int {
	out := make([]int, len(prs))
	for i, pr := range prs {
		out[i] = pr.Number
	}
	return out
}

func itoaAll(nums []int) []string {
	out := make([]string, len(nums))
	for i, n := range nums {
		out[i] = itoa(n)
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
