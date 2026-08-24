package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gsimone/dango-tui/internal/app"
	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
	"github.com/gsimone/dango-tui/internal/live"
	"github.com/gsimone/dango-tui/internal/summary"
)

type Options struct {
	StoryID  string
	Repo     string
	Provider summary.Provider
	Describe string
	Width    int
	Height   int
	Fetch    live.FetchFunc
	// EnrichCI fills CI after first paint. Nil is a no-op unless Fetch
	// is also nil (production then uses live.EnrichCI).
	EnrichCI live.EnrichCIFunc
	// Summarize is the provider hook. Nil uses summary.Run. Tests inject a fake.
	Summarize summary.Func
}

type Model struct {
	Width       int
	Height      int
	StoryIndex  int
	State       app.State
	Help        bool
	quitting    bool
	Fetching    bool
	Fetched     string
	feedbackSeq int
	fetchSeq    int
	LogoDots    [3]string
	Repo        string
	Provider    summary.Provider
	Describe    string
	Live        bool
	File        bool
	file        string
	stacks      []domain.Stack
	cacheState  data.CacheState
	fetchErr    error
	fetchedAt   time.Time
	fetch       live.FetchFunc
	enrichCI    live.EnrichCIFunc
	summarize   summary.Func
	summaryBusy bool
	summaryDone map[string]bool
	splashKeep  string
}

func New(opts Options) Model {
	width, height := opts.Width, opts.Height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	m := Model{
		Width:     width,
		Height:    height,
		State:     app.InitialState(),
		LogoDots:  domain.ProcessLogoDots(),
		Provider:  opts.Provider,
		Describe:  strings.TrimSpace(opts.Describe),
		fetch:     opts.Fetch,
		enrichCI:  opts.EnrichCI,
		summarize: opts.Summarize,
	}
	if m.fetch == nil {
		m.fetch = live.Fetch
		if m.enrichCI == nil {
			m.enrichCI = live.EnrichCI
		}
	}
	if m.summarize == nil {
		m.summarize = summary.Run
	}

	repo := strings.TrimSpace(opts.Repo)
	if opts.StoryID != "" {
		m.loadFixture(opts.StoryID)
		return m
	}
	if data.IsStackFile(repo) {
		m.loadFile(repo)
		return m
	}
	if repo == "" {
		return m
	}

	m.Live = true
	m.Repo = repo
	m.Fetching = true
	return m
}

func (m *Model) loadFile(path string) {
	m.File = true
	m.file = path
	m.Fetched = "last fetched 2 mins ago"
	repo, stacks, err := data.LoadStacks(path)
	if repo != "" {
		m.Repo = repo
	}
	if err != nil {
		m.fetchErr = err
		m.cacheState = data.CacheError
		if m.Repo == "" {
			m.Repo = path
		}
		return
	}
	m.fetchErr = nil
	m.stacks = stacks
	m.cacheState = data.CacheCurrent
}

func (m *Model) loadFixture(storyID string) {
	m.Fetched = "last fetched 2 mins ago"
	stories := data.Stories()
	idx := 0
	for i, story := range stories {
		if story.ID == storyID {
			idx = i
			break
		}
	}
	story := stories[idx]
	m.StoryIndex = idx
	m.stacks = story.Stacks
	m.cacheState = story.CacheState
}

func (m Model) Init() tea.Cmd {
	if m.Live {
		return m.fetchCmd(m.fetchSeq)
	}
	return m.startSummaries()
}

func (m Model) fetchCmd(token int) tea.Cmd {
	repo := m.Repo
	fetch := m.fetch
	return func() tea.Msg {
		stacks, err := fetch(repo)
		return fetchDoneMsg{stacks: stacks, err: err, at: time.Now(), token: token, live: true}
	}
}

func (m Model) waiting() bool {
	return m.Live && m.fetchErr == nil && len(m.stacks) == 0 && m.fetchedAt.IsZero()
}

func (m Model) showError() bool {
	return m.fetchErr != nil && len(m.stacks) == 0
}

func (m Model) errorCopyText() string {
	if m.fetchErr == nil {
		return ""
	}
	return strings.TrimSpace(m.fetchErr.Error())
}

func (m Model) splashCopyText() string {
	body := ""
	if text := m.errorCopyText(); text != "" {
		body = text
	} else if len(live.LastGHArgv) > 0 {
		body = live.FormatGHArgv(live.LastGHArgv)
	} else {
		repo := m.Repo
		if repo == "" {
			repo = "archetype-labs/app"
		}
		body = live.FormatGHArgv(live.PRListArgs(repo))
	}
	if sha := m.splashSHA(); sha != "" {
		if body != "" {
			return body + "\n" + sha
		}
		return sha
	}
	return body
}

func (m *Model) startSummaries() tea.Cmd {
	return m.startSelectedSummary()
}

func (m *Model) startSelectedSummary() tea.Cmd {
	if !m.Live || m.summaryBusy {
		return nil
	}
	if strings.TrimSpace(m.Describe) == "" && m.Provider.Empty() {
		return nil
	}
	stack, ok := m.SelectedStack()
	if !ok || stack.ID == "" {
		return nil
	}
	if strings.TrimSpace(stack.Description) != "" || m.summaryDone[stack.ID] {
		return nil
	}
	if m.summaryDone == nil {
		m.summaryDone = map[string]bool{}
	}
	m.summaryBusy = true
	token := m.fetchSeq
	job := summary.Job{Provider: m.Provider, Describe: m.Describe, Stack: stack, ID: stack.ID}
	run := m.summarize
	return func() tea.Msg {
		res := run(job)
		return summaryDoneMsg{
			token:       token,
			id:          res.ID,
			title:       res.Title,
			description: res.Description,
		}
	}
}

func (m Model) startCIEnrich() tea.Cmd {
	if !m.Live || m.enrichCI == nil {
		return nil
	}
	token := m.fetchSeq
	repo := m.Repo
	snap := append([]domain.Stack(nil), m.stacks...)
	for i := range snap {
		snap[i].PRs = append([]domain.PullRequest(nil), snap[i].PRs...)
	}
	enrich := m.enrichCI
	return func() tea.Msg {
		return ciDoneMsg{token: token, stacks: enrich(repo, snap)}
	}
}

func (m *Model) afterFetch() tea.Cmd {
	return tea.Batch(m.startSummaries(), m.startCIEnrich())
}

func (m Model) fetchBadge() string {
	if m.Fetching {
		return "⠋"
	}
	if m.Fetched == "" {
		return "last fetched 2 mins ago"
	}
	return m.Fetched
}

func relativeFetched(at, now time.Time) string {
	if at.IsZero() {
		return ""
	}
	mins := int(now.Sub(at).Minutes())
	if mins <= 0 {
		return "last fetched just now"
	}
	if mins == 1 {
		return "last fetched 1 min ago"
	}
	return "last fetched " + itoa(mins) + " mins ago"
}

func (m Model) repoLabel() string {
	if m.Repo != "" {
		return m.Repo
	}
	return "org/reponame"
}

func (m Model) Story() data.FixtureStory {
	stories := data.Stories()
	if m.Live || m.File || m.StoryIndex < 0 || m.StoryIndex >= len(stories) {
		return data.FixtureStory{Stacks: m.stacks, CacheState: m.cacheState}
	}
	return stories[m.StoryIndex]
}

func (m Model) Stacks() []domain.Stack {
	return app.FilterStacks(m.stacks, m.State.Query)
}

func (m Model) SelectedStack() (domain.Stack, bool) {
	stacks := m.Stacks()
	sel := app.ClampSelection(m.State.Selection, stacks)
	if len(stacks) == 0 || sel.StackIndex >= len(stacks) {
		return domain.Stack{}, false
	}
	return stacks[sel.StackIndex], true
}

func (m Model) SelectedPR() (domain.PullRequest, bool) {
	stack, ok := m.SelectedStack()
	if !ok {
		return domain.PullRequest{}, false
	}
	sel := app.ClampSelection(m.State.Selection, m.Stacks())
	if sel.PRIndex < 0 || sel.PRIndex >= len(stack.PRs) {
		return domain.PullRequest{}, false
	}
	return stack.PRs[sel.PRIndex], true
}

func (m *Model) choose(next app.Selection, reveal bool) {
	m.State.Selection = app.ClampSelection(next, m.Stacks())
	if reveal {
		m.State.CardVisible = true
	}
}

func (m *Model) checkout(pr domain.PullRequest) {
	m.State.Feedback = "Checked out " + pr.Branch + " · fixture simulation"
	m.State.CardVisible = true
}

type clearFeedbackMsg struct{ token int }

func (m *Model) copyBranch(pr domain.PullRequest) tea.Cmd {
	if pr.Branch == "" {
		m.State.Feedback = "No branch on #" + itoa(pr.Number)
		return m.clearFeedback()
	}
	copyText(pr.Branch)
	m.State.Feedback = "copied " + pr.Branch
	return m.clearFeedback()
}

func (m *Model) copyError() tea.Cmd {
	return m.copySplash()
}

func (m *Model) copySplash() tea.Cmd {
	text := m.splashCopyText()
	err := copyText(text)
	m.State.Feedback = "copied"
	if err != nil {
		m.splashKeep = text
	}
	return m.clearFeedback()
}

func (m *Model) clearFeedback() tea.Cmd {
	m.feedbackSeq++
	token := m.feedbackSeq
	return tea.Tick(900*time.Millisecond, func(time.Time) tea.Msg {
		return clearFeedbackMsg{token: token}
	})
}

func (m *Model) clamp() {
	m.State.Selection = app.ClampSelection(m.State.Selection, m.Stacks())
}

func (m Model) sourceState() string {
	switch m.Story().CacheState {
	case data.CacheStale:
		return "fixture cache · stale (simulated)"
	case data.CacheError:
		return "fixture refresh failed · no cached stacks"
	default:
		return "fixture cache · current · no network"
	}
}

func (m Model) emptyMessage() string {
	if strings.TrimSpace(m.State.Query) != "" {
		return "No match."
	}
	if m.waiting() || m.splash() {
		return ""
	}
	if m.fetchErr != nil {
		return ""
	}
	if m.cacheState == data.CacheError || m.Story().CacheState == data.CacheError {
		return "Refresh failed. No stacks are available."
	}
	return "No open stacks in this repository."
}

func (m Model) stackCount() int {
	return len(m.stacks)
}

func (m Model) layerCount() int {
	n := 0
	for _, stack := range m.stacks {
		n += len(stack.PRs)
	}
	return n
}

func (m Model) title() string {
	return "STACKS"
}

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

func deleteLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
