package cmd

import "github.com/getfider/fider/app/models/entity"

// MageOSReviewAddCandidate adds a board idea (PostID) or a manual initiative (Title) to the review list.
// Adding an idea that is already a candidate returns the existing candidate.
type MageOSReviewAddCandidate struct {
	PostID    int
	Title     string
	SourceURL string
	Track     string
	IsSample  bool

	Result  int
	Existed bool
}

// MageOSReviewUpdateCandidate sets the given fields (see mageos.ReviewCandidateFields).
type MageOSReviewUpdateCandidate struct {
	ID     int
	Fields map[string]string
}

// MageOSReviewSetAssessment records the current user's rating of one criterion.
type MageOSReviewSetAssessment struct {
	CandidateID int
	Criterion   string
	Rating      string
	Note        string
}

type MageOSReviewCreateMeeting struct {
	HeldOn   string
	Purpose  string
	IsSample bool

	Result int
}

type MageOSReviewUpdateMeeting struct {
	ID      int
	HeldOn  string
	Purpose string
}

// MageOSReviewAddItem puts a candidate on a meeting agenda. Adding it twice is a no-op.
type MageOSReviewAddItem struct {
	MeetingID   int
	CandidateID int

	Result int
}

// MageOSReviewRemoveItem takes a candidate off an agenda. Finalized decisions cannot be removed.
type MageOSReviewRemoveItem struct {
	ItemID int

	Refused bool
}

// MageOSReviewUpdateItem edits a draft decision. Finalized decisions must be reopened first.
type MageOSReviewUpdateItem struct {
	ItemID int
	Fields map[string]string

	Refused bool
}

// MageOSReviewFinalizeItem marks a decision final and records who finalized it.
// Missing lists what blocked finalizing (outcome, rationale).
type MageOSReviewFinalizeItem struct {
	ItemID int

	Missing []string
}

// MageOSReviewReopenItem turns a final decision back into a draft.
type MageOSReviewReopenItem struct {
	ItemID int
}

type MageOSReviewAddAction struct {
	CandidateID int
	ItemID      int
	Owner       string
	NextStep    string
	ReviewOn    string

	Result *entity.MageOSReviewAction
}

// MageOSReviewUpdateAction sets owner, nextStep, reviewOn or done ("true"/"false").
type MageOSReviewUpdateAction struct {
	ID     int
	Fields map[string]string
}

// MageOSReviewLogPublicStatus records that a team member changed the idea's public status from the review tool.
type MageOSReviewLogPublicStatus struct {
	CandidateID int
	Status      string
	Text        string
}

// MageOSReviewRemoveSampleData deletes every sample candidate and meeting.
type MageOSReviewRemoveSampleData struct {
	Removed int
}
