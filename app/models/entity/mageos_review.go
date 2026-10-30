package entity

import (
	"encoding/json"
	"time"

	"github.com/getfider/fider/app/models/enum"
)

// MageOSReviewIdea is the live board data of a candidate linked to an idea.
type MageOSReviewIdea struct {
	ID              int             `json:"id"`
	Number          int             `json:"number"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	Description     string          `json:"description"`
	Status          enum.PostStatus `json:"status"`
	Votes           int             `json:"votes"`
	HistoricalVotes int             `json:"historicalVotes"`
	Tags            []string        `json:"tags"`
	IsApproved      bool            `json:"isApproved"`
}

// MageOSReviewCandidate is an idea or initiative under consideration by the roadmap team.
type MageOSReviewCandidate struct {
	ID             int               `json:"id"`
	Idea           *MageOSReviewIdea `json:"idea,omitempty"`
	Title          string            `json:"title"`
	SourceURL      string            `json:"sourceUrl"`
	Track          string            `json:"track"`
	Stage          string            `json:"stage"`
	Delivery       string            `json:"delivery"`
	Problem        string            `json:"problem"`
	Beneficiaries  string            `json:"beneficiaries"`
	Outcome        string            `json:"outcome"`
	Scope          string            `json:"scope"`
	Exclusions     string            `json:"exclusions"`
	Evidence       string            `json:"evidence"`
	Dependencies   string            `json:"dependencies"`
	Risks          string            `json:"risks"`
	Effort         string            `json:"effort"`
	Owner          string            `json:"owner"`
	Reviewer       string            `json:"reviewer"`
	OperatingNeeds string            `json:"operatingNeeds"`
	OpenQuestions  string            `json:"openQuestions"`
	Links          string            `json:"links"`
	IsSample       bool              `json:"isSample"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	Missing        []string          `json:"missing"`
	LatestOutcome  string            `json:"latestOutcome"`
	LatestDecided  string            `json:"latestDecidedAt"`
	OpenActions    int               `json:"openActions"`
	Assessors      int               `json:"assessors"`
}

// MageOSReviewAssessment is one team member's rating of one criterion.
type MageOSReviewAssessment struct {
	CandidateID int       `json:"candidateId" db:"candidate_id"`
	UserID      int       `json:"userId" db:"user_id"`
	UserName    string    `json:"userName" db:"user_name"`
	Criterion   string    `json:"criterion" db:"criterion"`
	Rating      string    `json:"rating" db:"rating"`
	Note        string    `json:"note" db:"note"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

// MageOSReviewAction is a follow-up with an owner and an optional review date.
type MageOSReviewAction struct {
	ID             int       `json:"id" db:"id"`
	CandidateID    int       `json:"candidateId" db:"candidate_id"`
	CandidateTitle string    `json:"candidateTitle" db:"candidate_title"`
	IsSample       bool      `json:"isSample" db:"is_sample"`
	ItemID         int       `json:"itemId" db:"item_id"`
	Owner          string    `json:"owner" db:"owner"`
	NextStep       string    `json:"nextStep" db:"next_step"`
	ReviewOn       string    `json:"reviewOn" db:"review_on"`
	IsDone         bool      `json:"isDone" db:"is_done"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
}

// MageOSReviewMeeting is a roadmap meeting and its shortlist size.
type MageOSReviewMeeting struct {
	ID         int    `json:"id" db:"id"`
	HeldOn     string `json:"heldOn" db:"held_on"`
	Purpose    string `json:"purpose" db:"purpose"`
	IsSample   bool   `json:"isSample" db:"is_sample"`
	ItemCount  int    `json:"itemCount" db:"item_count"`
	FinalCount int    `json:"finalCount" db:"final_count"`
}

// MageOSReviewItem is one candidate on a meeting agenda and the decision recorded for it.
type MageOSReviewItem struct {
	ID                int    `json:"id" db:"id"`
	MeetingID         int    `json:"meetingId" db:"meeting_id"`
	MeetingHeldOn     string `json:"meetingHeldOn" db:"meeting_held_on"`
	CandidateID       int    `json:"candidateId" db:"candidate_id"`
	Position          int    `json:"position" db:"position"`
	DecisionRequested string `json:"decisionRequested" db:"decision_requested"`
	Notes             string `json:"notes" db:"notes"`
	Outcome           string `json:"outcome" db:"outcome"`
	Rationale         string `json:"rationale" db:"rationale"`
	Objections        string `json:"objections" db:"objections"`
	Unresolved        string `json:"unresolved" db:"unresolved"`
	Escalation        string `json:"escalation" db:"escalation"`
	FinalizedByName   string `json:"finalizedBy" db:"finalized_by_name"`
	FinalizedAt       string `json:"finalizedAt" db:"finalized_at"`

	// Filled by separate queries; dbx would treat a tagged pointer as a nested row.
	Candidate   *MageOSReviewCandidate    `json:"candidate,omitempty"`
	Assessments []*MageOSReviewAssessment `json:"assessments,omitempty"`
	Actions     []*MageOSReviewAction     `json:"actions,omitempty"`
}

// MageOSReviewEvent is one entry in the change history.
type MageOSReviewEvent struct {
	ID          int             `json:"id" db:"id"`
	CandidateID int             `json:"candidateId" db:"candidate_id"`
	MeetingID   int             `json:"meetingId" db:"meeting_id"`
	UserName    string          `json:"userName" db:"user_name"`
	Entity      string          `json:"entity" db:"entity"`
	EntityID    int             `json:"entityId" db:"entity_id"`
	Action      string          `json:"action" db:"action"`
	Changes     json.RawMessage `json:"changes" db:"changes"`
	CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
}

// MageOSReviewDashboard lists follow-ups that need attention.
type MageOSReviewDashboard struct {
	MissingInfo       []*MageOSReviewCandidate `json:"missingInfo"`
	UnassignedActions []*MageOSReviewAction    `json:"unassignedActions"`
	UpcomingReviews   []*MageOSReviewAction    `json:"upcomingReviews"`
	Overdue           []*MageOSReviewAction    `json:"overdue"`
	UpcomingMeetings  []*MageOSReviewMeeting   `json:"upcomingMeetings"`
}
