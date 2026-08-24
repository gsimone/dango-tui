package live

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gsimone/dango-tui/internal/domain"
)

// ErrGHMissing is the loud failure when the gh CLI is not on PATH.
var ErrGHMissing = errors.New("dango: gh CLI not found. Install https://cli.github.com and retry.")

// ErrGHAuth is a 401/403 (or gh auth) failure, not a flaky 502.
var ErrGHAuth = errors.New("dango: GitHub authentication or permission error. Check gh auth status.")

// FetchFunc loads stacks for owner/name. Tests inject a fake; production uses Fetch.
type FetchFunc func(repo string) ([]domain.Stack, error)

type runner func(args ...string) ([]byte, error)

var lookPath = exec.LookPath

// sleep is the backoff wait. Tests replace this.
var sleep = time.Sleep

// retryLimit is first try + one quick retry. Four GraphQL retries were the stall.
const retryLimit = 2

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

// LastGHArgv is the exact argv of the last gh invocation (no implicit extras).
// The paper error prints this so a 502 can be compared to the CLI one-liner.
var LastGHArgv []string

// FormatGHArgv is the exact command line dango runs.
func FormatGHArgv(args []string) string {
	if len(args) == 0 {
		return "gh"
	}
	return "gh " + strings.Join(args, " ")
}

func recordGHArgv(args []string) {
	LastGHArgv = append([]string(nil), args...)
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
	if isAuthMessage(msg) {
		return fmt.Errorf("%w %s", ErrGHAuth, msg)
	}
	return fmt.Errorf("%s: %s", FormatGHArgv(args), msg)
}

func classifyGHError(err error, args []string) error {
	if err == nil {
		return nil
	}
	if isGHNotFound(err) {
		return ErrGHMissing
	}
	if errors.Is(err, ErrGHAuth) || isAuthGH(err) {
		if errors.Is(err, ErrGHAuth) {
			return err
		}
		return fmt.Errorf("%w %s", ErrGHAuth, strings.TrimSpace(err.Error()))
	}
	if strings.HasPrefix(err.Error(), "gh ") {
		return err
	}
	return mapGHError(err, "", args)
}

func isAuthGH(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrGHAuth) {
		return true
	}
	return isAuthMessage(err.Error())
}

func isAuthMessage(msg string) bool {
	s := strings.ToLower(msg)
	if isTransientMessage(s) {
		return false
	}
	for _, needle := range []string{
		"http 401", "http 403", " 401:", " 403:",
		"unauthorized", "forbidden", "bad credentials",
		"auth login", "authentication", "permission denied",
		"resource not accessible", "must be authenticated",
	} {
		if strings.Contains(s, needle) {
			return true
		}
	}
	return false
}

func isTransientGH(err error) bool {
	if err == nil || isGHNotFound(err) || isAuthGH(err) {
		return false
	}
	return isTransientMessage(err.Error())
}

func isTransientMessage(msg string) bool {
	s := strings.ToLower(msg)
	return strings.Contains(s, "502") ||
		strings.Contains(s, "503") ||
		strings.Contains(s, "bad gateway") ||
		strings.Contains(s, "service unavailable")
}

func retryBackoff(attempt int) time.Duration {
	_ = attempt
	return 50 * time.Millisecond
}

func withRetry(run runner) runner {
	return func(args ...string) ([]byte, error) {
		var last error
		var lastOut []byte
		for attempt := 0; attempt < retryLimit; attempt++ {
			out, err := run(args...)
			if err == nil {
				return out, nil
			}
			if !isTransientGH(err) {
				return nil, err
			}
			last, lastOut = err, out
			if attempt+1 < retryLimit {
				sleep(retryBackoff(attempt))
			}
		}
		return lastOut, last
	}
}

// ghCommand builds `gh` so the process environment is inherited.
// Do not set cmd.Env — that would drop GH_TOKEN / GH_HOST.
func ghCommand(args ...string) *exec.Cmd {
	return exec.Command("gh", args...)
}

func runGH(args ...string) ([]byte, error) {
	recordGHArgv(args)
	cmd := ghCommand(args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, mapGHError(err, stderr.String(), args)
	}
	return out, nil
}

func prListArgs(repo string) []string {
	return []string{"pr", "list", "--repo", repo, "--state", "open", "--limit", "100",
		"--json", strings.Join(prListFields, ",")}
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
		return nil, fmt.Errorf("pass --repo archetype-labs/app")
	}
	inner := run
	run = func(args ...string) ([]byte, error) {
		recordGHArgv(args)
		return inner(args...)
	}
	run = withRetry(run)

	args := prListArgs(repo)
	raw, err := run(args...)
	if isGHNotFound(err) {
		return nil, ErrGHMissing
	}
	if err != nil {
		return nil, classifyGHError(err, args)
	}
	var listed []ghPR
	if decErr := json.Unmarshal(raw, &listed); decErr != nil {
		return nil, fmt.Errorf("decode gh pr list: %w", decErr)
	}

	prs := make([]RemotePR, 0, len(listed))
	for _, item := range listed {
		prs = append(prs, item.toRemote())
	}
	applyAuthorColors(prs)
	return GroupStacks(prs, "main"), nil
}

// prListFields is the first-paint grouping set. No body, no check rollup,
// no review history — those 502 GraphQL on a normal private repo.
var prListFields = []string{
	"number", "title", "url", "headRefName", "baseRefName",
	"author", "labels", "isDraft", "state",
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
