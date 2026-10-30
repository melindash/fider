package postgres_test

import (
	"testing"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
)

func TestMageOSReview_AddingAnIdeaTwiceKeepsOneCandidate(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	post := &cmd.AddNewPost{Title: "Faster admin grid", Description: "Slow with many SKUs"}
	bus.MustDispatch(jonSnowCtx, post)
	bus.MustDispatch(aryaStarkCtx, &cmd.AddVote{Post: post.Result, User: aryaStark})

	first := &cmd.MageOSReviewAddCandidate{PostID: post.Result.ID}
	second := &cmd.MageOSReviewAddCandidate{PostID: post.Result.ID}
	bus.MustDispatch(jonSnowCtx, first, second)
	Expect(first.Existed).IsFalse()
	Expect(second.Existed).IsTrue()
	Expect(second.Result).Equals(first.Result)

	list := &query.MageOSReviewListCandidates{}
	bus.MustDispatch(jonSnowCtx, list)
	Expect(list.Result).HasLen(1)
	c := list.Result[0]
	Expect(c.Title).Equals("Faster admin grid")
	Expect(c.Idea.Number).Equals(post.Result.Number)
	Expect(c.Idea.Votes).Equals(1)
	Expect(c.Missing).HasLen(9)
}

func TestMageOSReview_UnknownEffortIsNotMissing(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	add := &cmd.MageOSReviewAddCandidate{Title: "Contributor onboarding guide", Track: "contributor_experience"}
	bus.MustDispatch(jonSnowCtx, add)
	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewUpdateCandidate{ID: add.Result, Fields: map[string]string{"effort": "unknown", "owner": "Docs team"}})

	get := &query.MageOSReviewGetCandidate{ID: add.Result}
	bus.MustDispatch(jonSnowCtx, get)
	Expect(get.Result.Missing).Equals([]string{"problem", "beneficiaries", "outcome", "scope", "evidence", "reviewer", "operatingNeeds"})
}

func TestMageOSReview_RejectsUnknownFieldsAndValues(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	add := &cmd.MageOSReviewAddCandidate{Title: "Manual"}
	bus.MustDispatch(jonSnowCtx, add)
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MageOSReviewUpdateCandidate{ID: add.Result, Fields: map[string]string{"is_sample": "true"}})).IsNotNil()
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MageOSReviewUpdateCandidate{ID: add.Result, Fields: map[string]string{"stage": "approved"}})).IsNotNil()
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MageOSReviewSetAssessment{CandidateID: add.Result, Criterion: "benefit", Rating: "10"})).IsNotNil()
}

func TestMageOSReview_MeetingWorkflow(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	add := &cmd.MageOSReviewAddCandidate{Title: "Community meetup kit", Track: "community"}
	bus.MustDispatch(jonSnowCtx, add)
	id := add.Result

	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewSetAssessment{CandidateID: id, Criterion: "benefit", Rating: "strong", Note: "Asked for at every event"})
	bus.MustDispatch(aryaStarkCtx, &cmd.MageOSReviewSetAssessment{CandidateID: id, Criterion: "benefit", Rating: "unknown"})

	meeting := &cmd.MageOSReviewCreateMeeting{HeldOn: "2026-11-05", Purpose: "November priorities"}
	bus.MustDispatch(jonSnowCtx, meeting)
	item := &cmd.MageOSReviewAddItem{MeetingID: meeting.Result, CandidateID: id}
	again := &cmd.MageOSReviewAddItem{MeetingID: meeting.Result, CandidateID: id}
	bus.MustDispatch(jonSnowCtx, item, again)
	Expect(again.Result).Equals(item.Result)

	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewUpdateItem{ItemID: item.Result, Fields: map[string]string{"outcome": "prioritize"}})
	finalize := &cmd.MageOSReviewFinalizeItem{ItemID: item.Result}
	bus.MustDispatch(jonSnowCtx, finalize)
	Expect(finalize.Missing).Equals([]string{"rationale"})

	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewUpdateItem{ItemID: item.Result, Fields: map[string]string{"rationale": "High demand, kit is small", "unresolved": "Who prints materials?"}})
	finalize = &cmd.MageOSReviewFinalizeItem{ItemID: item.Result}
	bus.MustDispatch(jonSnowCtx, finalize)
	Expect(finalize.Missing).HasLen(0)

	locked := &cmd.MageOSReviewUpdateItem{ItemID: item.Result, Fields: map[string]string{"outcome": "decline"}}
	bus.MustDispatch(jonSnowCtx, locked)
	Expect(locked.Refused).IsTrue()

	action := &cmd.MageOSReviewAddAction{CandidateID: id, ItemID: item.Result, NextStep: "Draft the kit outline", ReviewOn: "2026-11-19"}
	bus.MustDispatch(jonSnowCtx, action)
	Expect(action.Result.Owner).Equals("")

	dashboard := &query.MageOSReviewGetDashboard{Today: "2026-11-10"}
	bus.MustDispatch(jonSnowCtx, dashboard)
	Expect(dashboard.Result.UnassignedActions).HasLen(1)
	Expect(dashboard.Result.UpcomingReviews).HasLen(1)
	Expect(dashboard.Result.Overdue).HasLen(0)

	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewUpdateAction{ID: action.Result.ID, Fields: map[string]string{"owner": "Arya"}})
	later := &query.MageOSReviewGetDashboard{Today: "2026-11-20"}
	bus.MustDispatch(jonSnowCtx, later)
	Expect(later.Result.UnassignedActions).HasLen(0)
	Expect(later.Result.Overdue).HasLen(1)

	got := &query.MageOSReviewGetMeeting{ID: meeting.Result}
	bus.MustDispatch(jonSnowCtx, got)
	Expect(got.Items).HasLen(1)
	Expect(got.Items[0].FinalizedByName).Equals("Jon Snow")
	Expect(got.Items[0].Candidate.Stage).Equals("decided")
	Expect(got.Items[0].Candidate.LatestOutcome).Equals("prioritize")
	Expect(got.Items[0].Candidate.Delivery).Equals("none")
	Expect(got.Items[0].Assessments).HasLen(2)
	Expect(got.Items[0].Actions).HasLen(1)

	history := &query.MageOSReviewGetCandidate{ID: id}
	bus.MustDispatch(jonSnowCtx, history)
	actions := []string{}
	for _, e := range history.Events {
		actions = append(actions, e.Entity+":"+e.Action)
	}
	Expect(actions).Equals([]string{
		"action:update", "action:create", "candidate:update", "item:finalize", "item:update", "item:update",
		"item:shortlist", "assessment:assess", "assessment:assess", "candidate:create",
	})

	reopen := &cmd.MageOSReviewReopenItem{ItemID: item.Result}
	bus.MustDispatch(aryaStarkCtx, reopen)
	remove := &cmd.MageOSReviewRemoveItem{ItemID: item.Result}
	bus.MustDispatch(jonSnowCtx, remove)
	Expect(remove.Refused).IsFalse()
}

func TestMageOSReview_RemoveSampleDataKeepsRealData(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	real := &cmd.MageOSReviewAddCandidate{Title: "Real initiative"}
	sample := &cmd.MageOSReviewAddCandidate{Title: "[Sample] Example", IsSample: true}
	bus.MustDispatch(jonSnowCtx, real, sample)
	meeting := &cmd.MageOSReviewCreateMeeting{HeldOn: "2026-11-05", Purpose: "Sample meeting", IsSample: true}
	bus.MustDispatch(jonSnowCtx, meeting)
	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewAddItem{MeetingID: meeting.Result, CandidateID: sample.Result})

	remove := &cmd.MageOSReviewRemoveSampleData{}
	bus.MustDispatch(jonSnowCtx, remove)
	Expect(remove.Removed).Equals(1)

	list := &query.MageOSReviewListCandidates{}
	meetings := &query.MageOSReviewListMeetings{}
	bus.MustDispatch(jonSnowCtx, list, meetings)
	Expect(list.Result).HasLen(1)
	Expect(list.Result[0].ID).Equals(real.Result)
	Expect(meetings.Result).HasLen(0)
}
