package live

import (
	"strings"

	"github.com/gsimone/dango-tui/internal/domain"
)

// RemotePR is one GitHub pull request as returned by gh, before stacking.
type RemotePR struct {
	Number           int
	Title            string
	URL              string
	HeadRefName      string
	BaseRefName      string
	HeadSHA          string
	Author           string
	AuthorColor      string
	AvatarURL        string
	Labels           []domain.Label
	Draft            bool
	Merged           bool
	Mergeable        string
	MergeQueueState  string
	ReviewDecision   string
	Approvals        int
	ChangesRequested bool
	CIState          string
	CIFailed         int
	CIPending        int
	CITotal          int
	Additions        int
	Deletions        int
	ChangedFiles     int
	Body             string
	StackNumber      int
	StackPosition    int
}

func ToDomain(pr RemotePR) domain.PullRequest {
	out := domain.PullRequest{
		Number:           pr.Number,
		Title:            pr.Title,
		URL:              pr.URL,
		Branch:           pr.HeadRefName,
		Author:           pr.Author,
		AuthorColor:      pr.AuthorColor,
		AvatarURL:        pr.AvatarURL,
		Labels:           append([]domain.Label(nil), pr.Labels...),
		Draft:            pr.Draft,
		Merged:           pr.Merged,
		MergeQueueState:  pr.MergeQueueState,
		ReviewDecision:   pr.ReviewDecision,
		Approvals:        pr.Approvals,
		ChangesRequested: pr.ChangesRequested,
		Additions:        pr.Additions,
		Deletions:        pr.Deletions,
		ChangedFiles:     pr.ChangedFiles,
		HeadSHA:          pr.HeadSHA,
		CI: domain.CISummary{
			State:   ciState(pr.CIState),
			Failed:  pr.CIFailed,
			Pending: pr.CIPending,
			Total:   pr.CITotal,
		},
	}
	switch strings.ToUpper(pr.Mergeable) {
	case "MERGEABLE":
		out.Mergeable = domain.MergeableTrue()
	case "CONFLICTING":
		out.Mergeable = domain.MergeableFalse()
	}
	if out.AuthorColor == "" {
		out.AuthorColor = domain.LoginColor(out.Author)
	}
	return out
}

func ciState(raw string) domain.CIState {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SUCCESS", "PASS", "PASSED":
		return domain.CISuccess
	case "FAILURE", "FAILED", "ERROR", "CANCELLED", "TIMED_OUT":
		return domain.CIFailure
	case "PENDING", "EXPECTED", "QUEUED", "IN_PROGRESS":
		return domain.CIPending
	default:
		return domain.CIUnknown
	}
}
