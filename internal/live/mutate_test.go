package live

import (
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestMutantGroupStacksEvidenceOrder(t *testing.T) {
	body := "Managed by Graphite.\n- #4\n- #5\n"
	prs := []RemotePR{
		{Number: 4, Title: "one", HeadRefName: "g/a", BaseRefName: "main", Body: body, StackNumber: 9, StackPosition: 1},
		{Number: 5, Title: "two", HeadRefName: "g/b", BaseRefName: "main", Body: body, StackNumber: 9, StackPosition: 2},
	}
	got := GroupStacks(prs, "main")
	if len(got) != 1 || got[0].ID != "gh-stack-9" {
		t.Fatalf("native wins over graphite, got %+v", ids(got))
	}

	graphiteFirst := func(in []RemotePR) []domain.Stack {
		g, rest := takeGraphite(in)
		var stacks []domain.Stack
		for _, group := range g {
			stacks = append(stacks, makeStack(group, "main", "graphite-first"))
		}
		n, rest := takeNative(rest)
		for _, group := range n {
			stacks = append(stacks, makeStack(group, "main", "native-late"))
		}
		_ = rest
		return stacks
	}
	if mutant := graphiteFirst(prs); len(mutant) > 0 && mutant[0].ID == "gh-stack-9" {
		t.Fatal("graphite-before-native mutant must not survive")
	}
}

func TestMutantGraphiteNeedsMarkerAndList(t *testing.T) {
	listed := graphiteNumbers("Managed by Graphite.\n- #4\n- #5\n- #6\n")
	if strings.Join(itoaAll(listed), ",") != "4,5,6" {
		t.Fatalf("real graphite list: %v", listed)
	}
	numbered := graphiteNumbers("Managed by Graphite.\n1. #10\n2. #11\n")
	if strings.Join(itoaAll(numbered), ",") != "10,11" {
		t.Fatalf("numbered graphite list: %v", numbered)
	}
	skipMarker := func(body string) []int {
		seen := map[int]bool{}
		var out []int
		for _, line := range strings.Split(body, "\n") {
			s := strings.TrimSpace(line)
			if !strings.HasPrefix(s, "-") {
				continue
			}
			s = strings.TrimSpace(s[1:])
			if !strings.HasPrefix(s, "#") {
				continue
			}
			n, ok := parseLeadingInt(s[1:])
			if !ok || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
		return out
	}
	plain := "- #1\n- #2\n"
	if len(graphiteNumbers(plain)) != 0 {
		t.Fatal("real parser requires a graphite marker")
	}
	if len(skipMarker(plain)) < 2 {
		t.Fatal("marker-skipping mutant must produce a false stack")
	}

	one := graphiteNumbers("Managed by Graphite.\n- #4\n")
	if len(one) != 1 || one[0] != 4 {
		t.Fatalf("single graphite number: %v", one)
	}
}

func TestMutantGhTitleAndCI(t *testing.T) {
	stack := domain.Stack{PRs: []domain.PullRequest{{Title: "  head  ", Branch: "gm/x"}}}
	if GhTitle(stack) != "head" {
		t.Fatalf("real title %q", GhTitle(stack))
	}
	preferBranch := func(s domain.Stack) string {
		if s.PRs[0].Branch != "" {
			return s.PRs[0].Branch
		}
		return GhTitle(s)
	}
	if preferBranch(stack) == "head" {
		t.Fatal("branch-first mutant must not survive")
	}

	if ciState("FAILURE") != domain.CIFailure {
		t.Fatal("real ci failure")
	}
	successOnly := func(raw string) domain.CIState {
		if strings.EqualFold(raw, "SUCCESS") {
			return domain.CISuccess
		}
		return domain.CIUnknown
	}
	if successOnly("FAILURE") == domain.CIFailure {
		t.Fatal("success-only ci mutant must not survive")
	}

	approvals, changes := reviewCounts([]ghReview{{State: "APPROVED"}, {State: "CHANGES_REQUESTED"}})
	if approvals != 1 || !changes {
		t.Fatalf("real reviews %d %v", approvals, changes)
	}
	ignoreChanges := func(latest []ghReview) (int, bool) {
		n := 0
		for _, rev := range latest {
			if strings.ToUpper(rev.State) == "APPROVED" {
				n++
			}
		}
		return n, false
	}
	if _, ch := ignoreChanges([]ghReview{{State: "CHANGES_REQUESTED"}}); ch {
		t.Fatal("ignore-changes mutant must not survive")
	}
}

func TestMutantGroupStacksDropsSingles(t *testing.T) {
	prs := []RemotePR{
		{Number: 1, Title: "solo", HeadRefName: "a", BaseRefName: "main"},
		{Number: 2, Title: "base", HeadRefName: "b", BaseRefName: "main"},
		{Number: 3, Title: "head", HeadRefName: "c", BaseRefName: "b"},
	}
	got := GroupStacks(prs, "main")
	if len(got) != 1 || len(got[0].PRs) != 2 {
		t.Fatalf("real grouping drops the 1-PR row, got %+v", ids(got))
	}
	keepAll := func(in []RemotePR) []domain.Stack {
		var stacks []domain.Stack
		stacks = append(stacks, chainStacks(in, "main")...)
		return stacks
	}
	if mutant := keepAll(prs); len(mutant) != 2 {
		t.Fatal("keep-all mutant must still produce a solo stack")
	}
	if KeepRealStacks([]domain.Stack{{PRs: []domain.PullRequest{{Number: 1}}}}) != nil {
		t.Fatal("KeepRealStacks must drop a one-ball stack")
	}
}

func TestMutantAuthorColorIsLoginStable(t *testing.T) {
	prs := []RemotePR{{Author: "gm", AvatarURL: "https://avatars.example/gm.png"}}
	applyAuthorColors(prs)
	if prs[0].AuthorColor != domain.LoginColor("gm") || domain.IsLowChromaHex(prs[0].AuthorColor) {
		t.Fatalf("real author color %q", prs[0].AuthorColor)
	}
	paintRed := func(in []RemotePR) {
		for i := range in {
			in[i].AuthorColor = "#c82828"
		}
	}
	clone := []RemotePR{{Author: "gm"}}
	paintRed(clone)
	if clone[0].AuthorColor == domain.LoginColor("gm") {
		t.Fatal("sampled-red mutant must not survive")
	}
}

func ids(stacks []domain.Stack) []string {
	out := make([]string, len(stacks))
	for i, s := range stacks {
		out[i] = s.ID
	}
	return out
}
