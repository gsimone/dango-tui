package data

import (
	"sync"

	"github.com/gsimone/dango-tui/internal/domain"
)

type PullRequestFixtureInput struct {
	State            domain.PrDisplayState
	Number           int
	Title            string
	URL              string
	Branch           string
	Author           string
	Labels           []domain.Label
	Draft            *bool
	Merged           *bool
	Mergeable        *bool
	MergeQueueState  *string
	ReviewDecision   *string
	Approvals        *int
	ChangesRequested *bool
	CI               *domain.CISummary
	Additions        *int
	Deletions        *int
	ChangedFiles     *int
	HeadSHA          *string
	UnsetMergeable   bool
}

var stateFlags = map[domain.PrDisplayState]func(*domain.PullRequest){
	domain.StateMerged: func(pr *domain.PullRequest) { pr.Merged = true },
	domain.StateDraft:  func(pr *domain.PullRequest) { pr.Draft = true },
	domain.StateCIFailure: func(pr *domain.PullRequest) {
		pr.CI = domain.CISummary{State: domain.CIFailure, Failed: 1, Pending: 0, Total: 9}
	},
	domain.StateReviewBlocked: func(pr *domain.PullRequest) {
		pr.ChangesRequested = true
		pr.ReviewDecision = "CHANGES_REQUESTED"
	},
	domain.StateQueued: func(pr *domain.PullRequest) {
		pr.MergeQueueState = "QUEUED"
		pr.ReviewDecision = "APPROVED"
		pr.CI = domain.CISummary{State: domain.CISuccess, Failed: 0, Pending: 0, Total: 9}
	},
	domain.StateReady: func(pr *domain.PullRequest) {
		pr.ReviewDecision = "APPROVED"
		pr.Approvals = 2
		pr.CI = domain.CISummary{State: domain.CISuccess, Failed: 0, Pending: 0, Total: 9}
	},
	domain.StateOpen: func(pr *domain.PullRequest) {
		pr.CI = domain.CISummary{State: domain.CIPending, Failed: 0, Pending: 2, Total: 9}
	},
}

func PRFixture(input PullRequestFixtureInput) domain.PullRequest {
	state := input.State
	if state == "" {
		state = domain.StateOpen
	}
	number := input.Number
	if number == 0 {
		number = 100
	}
	pr := domain.PullRequest{
		Number:       number,
		Title:        "Improve PR layer " + itoa(number),
		URL:          "https://github.com/example/stacks/pull/" + itoa(number),
		Branch:       "gm/stacks-" + itoa(number),
		Author:       "gianni",
		Draft:        false,
		Merged:       false,
		Mergeable:    domain.MergeableTrue(),
		Approvals:    0,
		Additions:    43,
		Deletions:    12,
		ChangedFiles: 4,
		HeadSHA:      "fixture" + padHex(number, 8),
		CI:           domain.CISummary{State: domain.CIUnknown, Failed: 0, Pending: 0, Total: 0},
	}
	if apply, ok := stateFlags[state]; ok {
		apply(&pr)
	}
	if input.Title != "" {
		pr.Title = input.Title
	}
	if input.URL != "" {
		pr.URL = input.URL
	}
	if input.Branch != "" {
		pr.Branch = input.Branch
	}
	if input.Author != "" {
		pr.Author = input.Author
	}
	if input.Draft != nil {
		pr.Draft = *input.Draft
	}
	if input.Merged != nil {
		pr.Merged = *input.Merged
	}
	if input.UnsetMergeable {
		pr.Mergeable = nil
	} else if input.Mergeable != nil {
		pr.Mergeable = input.Mergeable
	}
	if input.MergeQueueState != nil {
		pr.MergeQueueState = *input.MergeQueueState
	}
	if input.ReviewDecision != nil {
		pr.ReviewDecision = *input.ReviewDecision
	}
	if input.Approvals != nil {
		pr.Approvals = *input.Approvals
	}
	if input.ChangesRequested != nil {
		pr.ChangesRequested = *input.ChangesRequested
	}
	if input.Additions != nil {
		pr.Additions = *input.Additions
	}
	if input.Deletions != nil {
		pr.Deletions = *input.Deletions
	}
	if input.ChangedFiles != nil {
		pr.ChangedFiles = *input.ChangedFiles
	}
	if input.HeadSHA != nil {
		pr.HeadSHA = *input.HeadSHA
	}
	if input.CI != nil {
		pr.CI = *input.CI
	}
	if len(input.Labels) > 0 {
		pr.Labels = append([]domain.Label(nil), input.Labels...)
	}
	pr.AuthorColor = domain.LoginColor(pr.Author)
	return pr
}

type StackFixtureInput struct {
	ID          string
	Number      int
	Name        string
	BaseRef     string
	PRs         []domain.PullRequest
	Description *string
}

func StackFixture(input StackFixtureInput) domain.Stack {
	number := input.Number
	if number == 0 {
		number = 1
	}
	id := input.ID
	if id == "" {
		id = "fixture-stack-" + itoa(number)
	}
	name := input.Name
	if name == "" {
		name = "stack " + itoa(number)
	}
	baseRef := input.BaseRef
	if baseRef == "" {
		baseRef = "main"
	}
	prs := input.PRs
	if prs == nil {
		prs = []domain.PullRequest{PRFixture(PullRequestFixtureInput{Number: number * 100})}
	}
	description := "A deterministic fixture stack"
	if input.Description != nil {
		description = *input.Description
	}
	return domain.Stack{
		ID:          id,
		Number:      number,
		Name:        name,
		BaseRef:     baseRef,
		PRs:         prs,
		Description: description,
	}
}

type CacheState string

const (
	CacheCurrent CacheState = "current"
	CacheStale   CacheState = "stale"
	CacheError   CacheState = "error"
)

type FixtureStory struct {
	ID         string
	Label      string
	Stacks     []domain.Stack
	CacheState CacheState
}

func pr(number int, title string, state domain.PrDisplayState, extra ...PullRequestFixtureInput) domain.PullRequest {
	input := PullRequestFixtureInput{Number: number, Title: title, State: state}
	if len(extra) > 0 {
		over := extra[0]
		over.Number = number
		over.Title = title
		over.State = state
		if extra[0].Title != "" {
			over.Title = extra[0].Title
		}
		if extra[0].State != "" {
			over.State = extra[0].State
		}
		if extra[0].Number != 0 {
			over.Number = extra[0].Number
		}
		input = over
		input.Number = number
		input.Title = title
		input.State = state
	}
	return PRFixture(input)
}

func intPtr(v int) *int { return &v }

func desc(v string) *string { return &v }

// stories are the -story hook fixtures. Built on first use so a live
// fetch does not pay for them. Chaos and other unused stories stay out
// of the binary.
var (
	storiesOnce sync.Once
	stories     []FixtureStory
)

func Stories() []FixtureStory {
	storiesOnce.Do(func() {
		stories = []FixtureStory{
			{
				ID:    "mixed",
				Label: "mixed health",
				Stacks: []domain.Stack{
					StackFixture(StackFixtureInput{Number: 1, Name: "auth cleanup", Description: desc("Simplify authentication boundaries"), PRs: []domain.PullRequest{
						pr(184, "Split auth scope from session checks", domain.StateMerged, PullRequestFixtureInput{
							Labels: []domain.Label{
								{Name: "bug", Color: "#d73a4a"},
								{Name: "auth", Color: "#0e8a16"},
							},
						}),
						pr(185, "Keep service identity explicit", domain.StateReady),
						pr(186, "Remove implicit session fallback", domain.StateCIFailure),
					}}),
					StackFixture(StackFixtureInput{Number: 2, Name: "composer tokens", Description: desc("Add ontology tokens to email"), PRs: []domain.PullRequest{
						pr(211, "Add token catalogue", domain.StateReady),
						pr(212, "Map entity fields into composer", domain.StateReviewBlocked),
						pr(213, "Prepare token migration", domain.StateQueued),
					}}),
					StackFixture(StackFixtureInput{Number: 3, Name: "sync rewrite", Description: desc("Fix optimistic sync lifecycle"), PRs: []domain.PullRequest{
						pr(241, "Mark syncing work clearly", domain.StateDraft),
						pr(242, "Avoid duplicate invalidation", domain.StateOpen),
					}}),
				},
			},
			{ID: "all-merged", Label: "empty repository", Stacks: []domain.Stack{}},
			{
				ID:         "draft",
				Label:      "stale cache",
				CacheState: CacheStale,
				Stacks: []domain.Stack{
					StackFixture(StackFixtureInput{Number: 6, Name: "release notes", Description: desc("Cached 18m ago · waiting to refresh"), PRs: []domain.PullRequest{
						pr(321, "Draft release notes from merged work", domain.StateOpen, PullRequestFixtureInput{
							CI: &domain.CISummary{State: domain.CIUnknown, Failed: 0, Pending: 0, Total: 0},
						}),
						pr(322, "Publish the rollout note", domain.StateReady),
					}}),
				},
			},
			{ID: "ci-failing", Label: "refresh error", CacheState: CacheError, Stacks: []domain.Stack{}},
			{
				ID:    "changes-requested",
				Label: "changes requested",
				Stacks: []domain.Stack{
					StackFixture(StackFixtureInput{Number: 9, Name: "checkout safety", Description: desc("Needs one review follow-up"), PRs: []domain.PullRequest{
						pr(351, "Protect local checkout action", domain.StateReviewBlocked, PullRequestFixtureInput{
							Approvals:        intPtr(1),
							ChangesRequested: boolPtr(true),
						}),
					}}),
				},
			},
			{
				ID:    "large-stack",
				Label: "large stack",
				Stacks: []domain.Stack{
					StackFixture(StackFixtureInput{Number: 13, Name: "migration train", Description: desc("Ten independent layers, still one readable line"), PRs: largeStackPRs()}),
				},
			},
			{
				ID:    "pair",
				Label: "two layers",
				Stacks: []domain.Stack{
					StackFixture(StackFixtureInput{Number: 1, Name: "pair", Description: desc("Two reviewable cuts on the same checkout"), PRs: pairPRs()}),
				},
			},
			{
				ID:    "freight",
				Label: "twenty layers",
				Stacks: []domain.Stack{
					StackFixture(StackFixtureInput{Number: 2, Name: "freight train", Description: desc("Authored cutover train: mixed CI and review"), PRs: twentyLayerPRs()}),
				},
			},
		}
	})
	return stories
}

func pairPRs() []domain.PullRequest {
	return []domain.PullRequest{
		pr(701, "Land the checkout helper", domain.StateMerged),
		pr(702, "Prompt before the second hop", domain.StateOpen),
	}
}

func twentyLayerPRs() []domain.PullRequest {
	layers := []struct {
		title string
		state domain.PrDisplayState
	}{
		{"Land the schema cutover", domain.StateMerged},
		{"Backfill tenant rows", domain.StateMerged},
		{"Dual-write the reader", domain.StateMerged},
		{"Flip the default flag", domain.StateReady},
		{"Drain the old path", domain.StateReady},
		{"Queue the shadow drop", domain.StateQueued},
		{"Hold the reader cutover", domain.StateQueued},
		{"Drop the shadow table", domain.StateOpen},
		{"Document the cutover", domain.StateOpen},
		{"Draft the rollback note", domain.StateDraft},
		{"Remove the fallback client", domain.StateCIFailure},
		{"Restore the pager window", domain.StateReviewBlocked},
		{"Ship the cleanup note", domain.StateOpen},
		{"Pin the catalog checksum", domain.StateReady},
		{"Merge the leftover dual-write", domain.StateMerged},
		{"Watch the drain complete", domain.StateOpen},
		{"Draft the postmortem", domain.StateDraft},
		{"Fix the backfill checksum", domain.StateCIFailure},
		{"Queue the final drop", domain.StateQueued},
		{"Close the cutover ticket", domain.StateOpen},
	}
	out := make([]domain.PullRequest, len(layers))
	for i, layer := range layers {
		out[i] = pr(800+i, layer.title, layer.state)
	}
	return out
}

func largeStackPRs() []domain.PullRequest {
	states := []domain.PrDisplayState{
		domain.StateMerged, domain.StateMerged, domain.StateReady, domain.StateReady, domain.StateQueued,
		domain.StateOpen, domain.StateDraft, domain.StateCIFailure, domain.StateReviewBlocked, domain.StateOpen,
	}
	out := make([]domain.PullRequest, len(states))
	for i, state := range states {
		out[i] = pr(400+i, "Large stack layer "+itoa(i+1), state)
	}
	return out
}

func FixtureStoryIDs() []string {
	all := Stories()
	ids := make([]string, len(all))
	for i, story := range all {
		ids[i] = story.ID
	}
	return ids
}

func IsFixtureStoryID(value string) bool {
	for _, id := range FixtureStoryIDs() {
		if id == value {
			return true
		}
	}
	return false
}

func StoryByID(id string) FixtureStory {
	all := Stories()
	for _, story := range all {
		if story.ID == id {
			return story
		}
	}
	return all[0]
}

func boolPtr(v bool) *bool { return &v }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func padHex(n int, width int) string {
	const hexdigits = "0123456789abcdef"
	if n < 0 {
		n = 0
	}
	var buf [16]byte
	i := len(buf)
	if n == 0 {
		i--
		buf[i] = '0'
	}
	for n > 0 {
		i--
		buf[i] = hexdigits[n&0xf]
		n >>= 4
	}
	for len(buf)-i < width {
		i--
		buf[i] = '0'
	}
	return string(buf[i:])
}
