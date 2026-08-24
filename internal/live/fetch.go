package live

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// ErrGHMissing is the loud failure when the gh CLI is not on PATH.
var ErrGHMissing = errors.New("dango: gh CLI not found. Install https://cli.github.com and retry.")

// FetchFunc loads stacks for owner/name. Tests inject a fake; production uses Fetch.
type FetchFunc func(repo string) ([]domain.Stack, error)

type runner func(args ...string) ([]byte, error)

var lookPath = exec.LookPath

func requireGH() error {
	if _, err := lookPath("gh"); err != nil {
		return ErrGHMissing
	}
	return nil
}

func isGHNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrGHMissing) || errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var pathErr *exec.Error
	return errors.As(err, &pathErr) && errors.Is(pathErr.Err, exec.ErrNotFound)
}

func mapGHError(err error, stderr string, args []string) error {
	if err == nil {
		return nil
	}
	if isGHNotFound(err) {
		return ErrGHMissing
	}
	msg := strings.TrimSpace(stderr)
	if msg == "" {
		msg = err.Error()
	}
	n := min(len(args), 4)
	return fmt.Errorf("gh %s: %s", strings.Join(args[:n], " "), msg)
}

func runGH(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, mapGHError(err, stderr.String(), args)
	}
	return out, nil
}

// Fetch loads open PRs for repo via gh and groups them into stacks.
func Fetch(repo string) ([]domain.Stack, error) {
	if err := requireGH(); err != nil {
		return nil, err
	}
	return fetchWith(runGH, repo)
}

func fetchWith(run runner, repo string) ([]domain.Stack, error) {
	repo = strings.TrimSpace(repo)
	if repo == "" || !strings.Contains(repo, "/") {
		return nil, fmt.Errorf("pass --repo owner/name")
	}
	type call struct {
		raw []byte
		err error
	}
	var view, list, stacks call
	done := make(chan string, 3)
	go func() {
		view.raw, view.err = run("repo", "view", repo, "--json", "nameWithOwner,defaultBranchRef")
		done <- "view"
	}()
	go func() {
		list.raw, list.err = run("pr", "list", "--repo", repo, "--state", "open", "--limit", "200",
			"--json", strings.Join(prListFields, ","))
		done <- "list"
	}()
	go func() {
		stacks.raw, stacks.err = run("api", "repos/"+repo+"/stacks")
		done <- "stacks"
	}()
	for i := 0; i < 3; i++ {
		<-done
	}

	if isGHNotFound(view.err) || isGHNotFound(list.err) || isGHNotFound(stacks.err) {
		return nil, ErrGHMissing
	}
	defaultBranch := "main"
	if view.err == nil {
		var viewed ghRepo
		if json.Unmarshal(view.raw, &viewed) == nil {
			if viewed.DefaultBranchRef.Name != "" {
				defaultBranch = viewed.DefaultBranchRef.Name
			}
			if viewed.NameWithOwner != "" {
				repo = viewed.NameWithOwner
			}
		}
	}
	if list.err != nil {
		return nil, mapGHError(list.err, "", []string{"pr", "list"})
	}
	var listed []ghPR
	if err := json.Unmarshal(list.raw, &listed); err != nil {
		return nil, fmt.Errorf("decode gh pr list: %w", err)
	}

	prs := make([]RemotePR, 0, len(listed))
	for _, item := range listed {
		prs = append(prs, item.toRemote())
	}
	applyAuthorColors(prs)
	if stacks.err == nil {
		applyNativeStacks(stacks.raw, prs)
	}
	return GroupStacks(prs, defaultBranch), nil
}

var prListFields = []string{
	"number", "title", "url", "headRefName", "baseRefName", "headRefOid",
	"author", "labels", "isDraft", "state", "mergeable", "mergeStateStatus",
	"reviewDecision", "latestReviews",
	"additions", "deletions", "changedFiles", "body", "statusCheckRollup",
}

type ghRepo struct {
	NameWithOwner    string `json:"nameWithOwner"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

type ghActor struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatarUrl"`
}

type ghLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ghReview struct {
	State string `json:"state"`
}

type ghCheck struct {
	State      string `json:"state"`
	Conclusion string `json:"conclusion"`
	Status     string `json:"status"`
}

type ghPR struct {
	Number            int        `json:"number"`
	Title             string     `json:"title"`
	URL               string     `json:"url"`
	HeadRefName       string     `json:"headRefName"`
	BaseRefName       string     `json:"baseRefName"`
	HeadRefOid        string     `json:"headRefOid"`
	Author            ghActor    `json:"author"`
	Labels            []ghLabel  `json:"labels"`
	IsDraft           bool       `json:"isDraft"`
	State             string     `json:"state"`
	Mergeable         string     `json:"mergeable"`
	MergeStateStatus  string     `json:"mergeStateStatus"`
	ReviewDecision    string     `json:"reviewDecision"`
	LatestReviews     []ghReview `json:"latestReviews"`
	Additions         int        `json:"additions"`
	Deletions         int        `json:"deletions"`
	ChangedFiles      int        `json:"changedFiles"`
	Body              string     `json:"body"`
	StatusCheckRollup []ghCheck  `json:"statusCheckRollup"`
}

func (p ghPR) toRemote() RemotePR {
	approvals, changes := reviewCounts(p.LatestReviews)
	ciState, failed, pending, total := rollupCI(p.StatusCheckRollup)
	queue := ""
	if strings.EqualFold(p.MergeStateStatus, "QUEUED") {
		queue = "QUEUED"
	}
	labels := make([]domain.Label, 0, len(p.Labels))
	for _, lab := range p.Labels {
		name := strings.TrimSpace(lab.Name)
		if name == "" {
			continue
		}
		labels = append(labels, domain.Label{Name: name, Color: domain.NormalizeHex(lab.Color)})
	}
	return RemotePR{
		Number:           p.Number,
		Title:            p.Title,
		URL:              p.URL,
		HeadRefName:      p.HeadRefName,
		BaseRefName:      p.BaseRefName,
		HeadSHA:          p.HeadRefOid,
		Author:           p.Author.Login,
		AvatarURL:        strings.TrimSpace(p.Author.AvatarURL),
		Labels:           labels,
		Draft:            p.IsDraft,
		Merged:           strings.EqualFold(p.State, "MERGED"),
		Mergeable:        p.Mergeable,
		MergeQueueState:  queue,
		ReviewDecision:   p.ReviewDecision,
		Approvals:        approvals,
		ChangesRequested: changes || strings.EqualFold(p.ReviewDecision, "CHANGES_REQUESTED"),
		CIState:          ciState,
		CIFailed:         failed,
		CIPending:        pending,
		CITotal:          total,
		Additions:        p.Additions,
		Deletions:        p.Deletions,
		ChangedFiles:     p.ChangedFiles,
		Body:             p.Body,
	}
}

func reviewCounts(latest []ghReview) (approvals int, changes bool) {
	for _, rev := range latest {
		switch strings.ToUpper(rev.State) {
		case "APPROVED":
			approvals++
		case "CHANGES_REQUESTED":
			changes = true
		}
	}
	return approvals, changes
}

func rollupCI(checks []ghCheck) (state string, failed, pending, total int) {
	total = len(checks)
	if total == 0 {
		return "", 0, 0, 0
	}
	for _, check := range checks {
		token := strings.ToUpper(check.State)
		if token == "" {
			token = strings.ToUpper(check.Conclusion)
		}
		status := strings.ToUpper(check.Status)
		switch {
		case token == "FAILURE" || token == "ERROR" || token == "CANCELLED" || token == "TIMED_OUT" || token == "ACTION_REQUIRED":
			failed++
		case token == "PENDING" || token == "EXPECTED" || token == "QUEUED" || status == "IN_PROGRESS" || status == "QUEUED" || status == "PENDING":
			pending++
		}
	}
	switch {
	case failed > 0:
		return "FAILURE", failed, pending, total
	case pending > 0:
		return "PENDING", failed, pending, total
	default:
		return "SUCCESS", failed, pending, total
	}
}

type ghNativeStack struct {
	Number       int    `json:"number"`
	Size         int    `json:"size"`
	BaseRefName  string `json:"base_ref_name"`
	BaseRef      string `json:"baseRefName"`
	PullRequests []struct {
		Number   int `json:"number"`
		Position int `json:"position"`
	} `json:"pull_requests"`
	Entries []struct {
		Position    int `json:"position"`
		PullRequest struct {
			Number int `json:"number"`
		} `json:"pull_request"`
		Number int `json:"number"`
	} `json:"entries"`
}

func applyAuthorColors(prs []RemotePR) {
	cache := map[string]string{}
	for i := range prs {
		login := prs[i].Author
		if hex, ok := cache[login]; ok {
			prs[i].AuthorColor = hex
			continue
		}
		hex := domain.LoginColor(login)
		cache[login] = hex
		prs[i].AuthorColor = hex
	}
}

func applyNativeStacks(raw []byte, prs []RemotePR) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return
	}
	var stacks []ghNativeStack
	if json.Unmarshal(raw, &stacks) != nil || len(stacks) == 0 {
		return
	}
	byNum := map[int]*RemotePR{}
	for i := range prs {
		byNum[prs[i].Number] = &prs[i]
	}
	for _, stack := range stacks {
		if stack.Number == 0 {
			continue
		}
		type entry struct{ number, position int }
		var entries []entry
		for _, pr := range stack.PullRequests {
			entries = append(entries, entry{pr.Number, pr.Position})
		}
		for _, e := range stack.Entries {
			n := e.PullRequest.Number
			if n == 0 {
				n = e.Number
			}
			entries = append(entries, entry{n, e.Position})
		}
		for _, e := range entries {
			if pr, ok := byNum[e.number]; ok {
				pr.StackNumber = stack.Number
				pr.StackPosition = e.position
			}
		}
	}
}
