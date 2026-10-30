package mageos_test

import (
	"strings"
	"testing"

	"github.com/getfider/fider/app/models/entity"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/mageos"
)

func TestReviewMeetingSummary(t *testing.T) {
	RegisterT(t)

	meeting := &entity.MageOSReviewMeeting{HeldOn: "2026-11-05", Purpose: "November priorities"}
	items := []*entity.MageOSReviewItem{
		{
			Outcome: "prioritize", Rationale: "Strong demand <b>", FinalizedAt: "2026-11-05T18:00:00Z", FinalizedByName: "Jon Snow",
			Notes:     "internal discussion",
			Candidate: &entity.MageOSReviewCandidate{Title: "Faster grid", Delivery: "none", Idea: &entity.MageOSReviewIdea{Number: 7, Slug: "faster-grid", Votes: 12}},
			Actions:   []*entity.MageOSReviewAction{{NextStep: "Scope it", ReviewOn: "2026-11-19"}, {NextStep: "Done already", IsDone: true}},
		},
		{Outcome: "defer", Unresolved: "Who maintains it?", Candidate: &entity.MageOSReviewCandidate{Title: "Meetup kit", IsSample: true}},
	}

	html, text := mageos.ReviewMeetingSummary("https://ideas.example", meeting, items)

	Expect(strings.Contains(text, "1 of 2 decisions final.")).IsTrue()
	Expect(strings.Contains(text, "- Outcome: Prioritize (not scheduled). Final, recorded by Jon Snow")).IsTrue()
	Expect(strings.Contains(text, "https://ideas.example/posts/7/faster-grid (12 votes)")).IsTrue()
	Expect(strings.Contains(text, "  - Unassigned: Scope it (review 2026-11-19)")).IsTrue()
	Expect(strings.Contains(text, "Done already")).IsFalse()
	Expect(strings.Contains(text, "Meetup kit (sample)")).IsTrue()
	Expect(strings.Contains(text, "Defer. DRAFT, not final")).IsTrue()
	Expect(strings.Contains(text, "internal discussion")).IsFalse()
	Expect(strings.Contains(html, "Strong demand &lt;b&gt;")).IsTrue()
}
