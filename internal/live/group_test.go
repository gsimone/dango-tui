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
	if len(stacks) != 1 {
		t.Fatalf("solo is not a stack, got %d stacks", len(stacks))
	}
	chain := numbers(stacks[0].PRs)
	if strings.Join(itoaAll(chain), ",") != "10,11,12" {
		t.Fatalf("chain %v", chain)
	}
	if stackKey(stacks[0]) != 12 {
		t.Fatalf("chain key %d", stackKey(stacks[0]))
	}
}

func TestGroupStacksDropsOnePR(t *testing.T) {
	stacks := GroupStacks([]RemotePR{
		{Number: 20, Title: "solo", HeadRefName: "fix/one", BaseRefName: "main"},
	}, "main")
	if len(stacks) != 0 {
		t.Fatalf("1-PR group must not be a stack: %+v", stacks)
	}
	mixed := GroupStacks([]RemotePR{
		{Number: 1, Title: "base", HeadRefName: "a", BaseRefName: "main"},
		{Number: 2, Title: "head", HeadRefName: "b", BaseRefName: "a"},
		{Number: 9, Title: "other", HeadRefName: "solo", BaseRefName: "main"},
	}, "main")
	if len(mixed) != 1 || len(mixed[0].PRs) != 2 {
		t.Fatalf("keep the 2-PR chain only, got %+v", mixed)
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
	if len(stacks) != 1 {
		t.Fatalf("solo dropped, got %d", len(stacks))
	}
	if stacks[0].ID != "gh-stack-7" {
		t.Fatalf("native id %s", stacks[0].ID)
	}
	if strings.Join(itoaAll(numbers(stacks[0].PRs)), ",") != "1,2" {
		t.Fatalf("native %v", numbers(stacks[0].PRs))
	}
}

func TestGroupStacksDoesNotInventTitle(t *testing.T) {
	stacks := GroupStacks([]RemotePR{
		{Number: 1, Title: "base layer", HeadRefName: "a", BaseRefName: "main"},
		{Number: 2, Title: "head layer", HeadRefName: "b", BaseRefName: "a"},
	}, "main")
	if len(stacks) != 1 {
		t.Fatalf("got %d", len(stacks))
	}
	if stacks[0].Name != "base layer" {
		t.Fatalf("list paints the short gh name first, got %q", stacks[0].Name)
	}
	if stacks[0].Summary != "" {
		t.Fatalf("fetch must not invent a generated summary, got %q", stacks[0].Summary)
	}
}

func TestShortNameKeepsTicketInTheList(t *testing.T) {
	if got := ShortName("LEV-182: Bound hosts to the session"); got != "LEV-182" {
		t.Fatalf("ticket sentence: %q", got)
	}
	if got := ShortName("LEV-182 — Bound hosts"); got != "LEV-182" {
		t.Fatalf("em dash: %q", got)
	}
	if got := ShortName("auth cleanup"); got != "auth cleanup" {
		t.Fatalf("authored short name: %q", got)
	}
	if got := ShortName("LEV-182"); got != "LEV-182" {
		t.Fatalf("bare ticket: %q", got)
	}
	stamped := StampGhNames([]domain.Stack{{
		Name: "LEV-182: Bound hosts to the session",
		PRs:  []domain.PullRequest{{Title: "LEV-182: Bound hosts to the session"}, {Title: "head"}},
	}})
	if stamped[0].Name != "LEV-182" {
		t.Fatalf("stamp must clip the sentence, got %q", stamped[0].Name)
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
