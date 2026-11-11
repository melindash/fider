package postgres_test

import (
	"testing"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
)

func TestMageOSReview_VolunteersByAccountOrGitHub(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	add := &cmd.MageOSReviewAddCandidate{Title: "Admin grid performance"}
	bus.MustDispatch(jonSnowCtx, add)

	byAccount := &cmd.MageOSReviewAddVolunteer{CandidateID: add.Result, UserID: aryaStark.ID}
	byGitHub := &cmd.MageOSReviewAddVolunteer{CandidateID: add.Result, GitHubLogin: "@Mage-OS-Dev"}
	sameGitHub := &cmd.MageOSReviewAddVolunteer{CandidateID: add.Result, GitHubLogin: "mage-os-dev"}
	bus.MustDispatch(jonSnowCtx, byAccount, byGitHub, sameGitHub)

	Expect(byAccount.Result.UserName).Equals(aryaStark.Name)
	Expect(byGitHub.Result.GitHubLogin).Equals("Mage-OS-Dev")
	Expect(sameGitHub.Result.ID).Equals(byGitHub.Result.ID)

	get := &query.MageOSReviewGetCandidate{ID: add.Result}
	bus.MustDispatch(jonSnowCtx, get)
	Expect(get.Volunteers).HasLen(2)

	bus.MustDispatch(jonSnowCtx, &cmd.MageOSReviewRemoveVolunteer{ID: byAccount.Result.ID})
	get = &query.MageOSReviewGetCandidate{ID: add.Result}
	bus.MustDispatch(jonSnowCtx, get)
	Expect(get.Volunteers).HasLen(1)
	Expect(get.Volunteers[0].GitHubLogin).Equals("Mage-OS-Dev")
	Expect(get.Events[0].Entity).Equals("volunteer")
}

func TestMageOSReview_VolunteerNeedsAnIdentity(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	add := &cmd.MageOSReviewAddCandidate{Title: "Admin grid performance"}
	bus.MustDispatch(jonSnowCtx, add)

	Expect(bus.Dispatch(jonSnowCtx, &cmd.MageOSReviewAddVolunteer{CandidateID: add.Result})).IsNotNil()
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MageOSReviewAddVolunteer{CandidateID: add.Result, GitHubLogin: "not a login"})).IsNotNil()
	Expect(bus.Dispatch(jonSnowCtx, &cmd.MageOSReviewAddVolunteer{CandidateID: add.Result + 1000, GitHubLogin: "melindash"})).IsNotNil()
}
