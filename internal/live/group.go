package live

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// GroupStacks turns repo PRs into stacks. Order of evidence:
// GitHub native stack ids, Graphite body lists, then base-branch chains.
func GroupStacks(prs []RemotePR, defaultBranch string) []domain.Stack {
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if len(prs) == 0 {
		return nil
	}

	var stacks []domain.Stack
	native, rest := takeNative(prs)
	for _, group := range native {
		stacks = append(stacks, makeStack(group, defaultBranch, fmt.Sprintf("gh-stack-%d", group[0].StackNumber)))
	}
	graphite, rest := takeGraphite(rest)
	for _, group := range graphite {
		stacks = append(stacks, makeStack(group, defaultBranch, fmt.Sprintf("graphite-%d", group[0].Number)))
	}
	stacks = append(stacks, chainStacks(rest, defaultBranch)...)

	sort.SliceStable(stacks, func(i, j int) bool {
		return stackKey(stacks[i]) > stackKey(stacks[j])
	})
	stacks = KeepRealStacks(stacks)
	for i := range stacks {
		stacks[i].Number = i + 1
	}
	return StampGhNames(stacks)
}

// KeepRealStacks drops 1-PR groups. A stack is two or more PRs.
func KeepRealStacks(stacks []domain.Stack) []domain.Stack {
	out := stacks[:0]
	for _, stack := range stacks {
		if len(stack.PRs) >= 2 {
			out = append(out, stack)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func stackKey(stack domain.Stack) int {
	max := 0
	for _, pr := range stack.PRs {
		if pr.Number > max {
			max = pr.Number
		}
	}
	return max
}

func makeStack(prs []RemotePR, defaultBranch, id string) domain.Stack {
	layers := make([]domain.PullRequest, len(prs))
	for i, pr := range prs {
		layers[i] = ToDomain(pr)
	}
	base := prs[0].BaseRefName
	if base == "" {
		base = defaultBranch
	}
	return domain.Stack{
		ID:      id,
		Name:    GhTitle(domain.Stack{PRs: layers}),
		BaseRef: base,
		PRs:     layers,
	}
}

// StampGhNames fills an empty list name from GitHub titles. It does not
// invent a generated summary.
func StampGhNames(stacks []domain.Stack) []domain.Stack {
	for i := range stacks {
		if strings.TrimSpace(stacks[i].Name) == "" {
			stacks[i].Name = GhTitle(stacks[i])
		}
	}
	return stacks
}

func GhTitle(stack domain.Stack) string {
	for _, pr := range stack.PRs {
		if t := strings.TrimSpace(pr.Title); t != "" {
			return t
		}
		if pr.Branch != "" {
			return pr.Branch
		}
	}
	return ""
}

func takeNative(prs []RemotePR) (groups [][]RemotePR, rest []RemotePR) {
	byNum := map[int][]RemotePR{}
	for _, pr := range prs {
		if pr.StackNumber > 0 {
			byNum[pr.StackNumber] = append(byNum[pr.StackNumber], pr)
			continue
		}
		rest = append(rest, pr)
	}
	nums := make([]int, 0, len(byNum))
	for n := range byNum {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		group := byNum[n]
		sort.SliceStable(group, func(i, j int) bool {
			if group[i].StackPosition != group[j].StackPosition {
				return group[i].StackPosition < group[j].StackPosition
			}
			return group[i].Number < group[j].Number
		})
		groups = append(groups, group)
	}
	return groups, rest
}

func takeGraphite(prs []RemotePR) (groups [][]RemotePR, rest []RemotePR) {
	byNum := map[int]RemotePR{}
	for _, pr := range prs {
		byNum[pr.Number] = pr
	}
	used := map[int]bool{}
	for _, pr := range prs {
		nums := graphiteNumbers(pr.Body)
		if len(nums) < 2 {
			continue
		}
		var group []RemotePR
		for _, n := range nums {
			p, ok := byNum[n]
			if !ok || used[n] {
				continue
			}
			group = append(group, p)
			used[n] = true
		}
		if len(group) >= 2 {
			groups = append(groups, group)
		} else {
			for _, p := range group {
				used[p.Number] = false
			}
		}
	}
	for _, pr := range prs {
		if !used[pr.Number] {
			rest = append(rest, pr)
		}
	}
	return groups, rest
}

func graphiteNumbers(body string) []int {
	if !strings.Contains(strings.ToLower(body), "graphite") {
		return nil
	}
	seen := map[int]bool{}
	var out []int
	for _, line := range strings.Split(body, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		if s[0] == '-' || s[0] == '*' {
			s = strings.TrimSpace(s[1:])
		} else {
			i := 0
			for i < len(s) && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			if i == 0 || i >= len(s) || s[i] != '.' {
				continue
			}
			s = strings.TrimSpace(s[i+1:])
		}
		if !strings.HasPrefix(s, "#") {
			continue
		}
		n, ok := parseLeadingInt(s[1:])
		if !ok || n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func parseLeadingInt(s string) (int, bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, false
	}
	n := 0
	for j := 0; j < i; j++ {
		n = n*10 + int(s[j]-'0')
	}
	return n, true
}

func chainStacks(prs []RemotePR, defaultBranch string) []domain.Stack {
	if len(prs) == 0 {
		return nil
	}
	byHead := map[string]RemotePR{}
	byNum := map[int]RemotePR{}
	for _, pr := range prs {
		byHead[pr.HeadRefName] = pr
		byNum[pr.Number] = pr
	}
	children := map[int][]RemotePR{}
	parentOf := map[int]int{}
	for _, pr := range prs {
		parent, ok := byHead[pr.BaseRefName]
		if !ok || parent.Number == pr.Number {
			continue
		}
		children[parent.Number] = append(children[parent.Number], pr)
		parentOf[pr.Number] = parent.Number
	}

	var queue []RemotePR
	for _, pr := range prs {
		if _, ok := parentOf[pr.Number]; !ok {
			queue = append(queue, pr)
		}
	}
	sort.SliceStable(queue, func(i, j int) bool { return queue[i].Number < queue[j].Number })

	used := map[int]bool{}
	var stacks []domain.Stack
	for i := 0; i < len(queue); i++ {
		root := queue[i]
		if used[root.Number] {
			continue
		}
		chain := []RemotePR{root}
		used[root.Number] = true
		cur := root
		for {
			var next []RemotePR
			for _, kid := range children[cur.Number] {
				if !used[kid.Number] {
					next = append(next, kid)
				}
			}
			if len(next) == 0 {
				break
			}
			sort.SliceStable(next, func(i, j int) bool { return next[i].Number < next[j].Number })
			pick := next[0]
			chain = append(chain, pick)
			used[pick.Number] = true
			cur = pick
			queue = append(queue, next[1:]...)
		}
		stacks = append(stacks, makeStack(chain, defaultBranch, fmt.Sprintf("stack-%d", chain[0].Number)))
	}
	return stacks
}
