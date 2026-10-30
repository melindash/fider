package query

import "github.com/getfider/fider/app/models/entity"

type MageOSReviewListCandidates struct {
	IncludeArchived bool

	Result []*entity.MageOSReviewCandidate
}

type MageOSReviewGetCandidate struct {
	ID int

	Result      *entity.MageOSReviewCandidate
	Assessments []*entity.MageOSReviewAssessment
	Actions     []*entity.MageOSReviewAction
	Items       []*entity.MageOSReviewItem
	Events      []*entity.MageOSReviewEvent
}

type MageOSReviewListMeetings struct {
	Result []*entity.MageOSReviewMeeting
}

// MageOSReviewGetMeeting returns a meeting with each agenda item's candidate, assessments and actions.
type MageOSReviewGetMeeting struct {
	ID int

	Result *entity.MageOSReviewMeeting
	Items  []*entity.MageOSReviewItem
	Events []*entity.MageOSReviewEvent
}

// MageOSReviewGetAssessments returns every assessment for the given candidates.
type MageOSReviewGetAssessments struct {
	CandidateIDs []int

	Result []*entity.MageOSReviewAssessment
}

type MageOSReviewGetDashboard struct {
	Today string

	Result *entity.MageOSReviewDashboard
}
