package domain

type CIState string

const (
	CISuccess CIState = "success"
	CIPending CIState = "pending"
	CIFailure CIState = "failure"
	CIUnknown CIState = "unknown"
)

type CISummary struct {
	State   CIState
	Failed  int
	Pending int
	Total   int
}

type Label struct {
	Name  string
	Color string // #rrggbb from GitHub, or empty
}

type PullRequest struct {
	Number           int
	Title            string
	URL              string
	Branch           string
	Author           string
	AuthorColor      string // ● ink; avatar sample or login-stable fallback
	AvatarURL        string
	Labels           []Label
	Draft            bool
	Merged           bool
	Mergeable        *bool
	MergeQueueState  string
	ReviewDecision   string
	Approvals        int
	ChangesRequested bool
	CI               CISummary
	Additions        int
	Deletions        int
	ChangedFiles     int
	HeadSHA          string
}

type Stack struct {
	ID          string
	Number      int
	Name        string
	BaseRef     string
	PRs         []PullRequest
	Description string
	Summary     string
}

type PrDisplayState string

const (
	StateMerged        PrDisplayState = "merged"
	StateDraft         PrDisplayState = "draft"
	StateCIFailure     PrDisplayState = "ci-failure"
	StateReviewBlocked PrDisplayState = "review-blocked"
	StateQueued        PrDisplayState = "queued"
	StateReady         PrDisplayState = "ready"
	StateOpen          PrDisplayState = "open"
)

func BoolPtr(v bool) *bool { return &v }

func MergeableTrue() *bool  { return BoolPtr(true) }
func MergeableFalse() *bool { return BoolPtr(false) }
