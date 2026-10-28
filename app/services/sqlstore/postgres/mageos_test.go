package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mageos"
)

func newAnonymousVoter(t *testing.T, token string) (*entity.MageOSAnonymousVoter, context.Context) {
	ensure := &cmd.MageOSEnsureAnonymousVoter{TokenHash: mageos.HashToken(token)}
	bus.MustDispatch(demoTenantCtx, ensure)
	return ensure.Result, context.WithValue(demoTenantCtx, mageos.VoterCtxKey, ensure.Result)
}

func reloadPost(ctx context.Context, number int) *entity.Post {
	get := &query.GetPostByNumber{Number: number}
	bus.MustDispatch(ctx, get)
	return get.Result
}

func TestMageOS_EnsureVoterIsIdempotent(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	first := &cmd.MageOSEnsureAnonymousVoter{TokenHash: mageos.HashToken("token-a")}
	bus.MustDispatch(demoTenantCtx, first)
	Expect(first.Created).IsTrue()

	second := &cmd.MageOSEnsureAnonymousVoter{TokenHash: mageos.HashToken("token-a")}
	bus.MustDispatch(demoTenantCtx, second)
	Expect(second.Created).IsFalse()
	Expect(second.Result.ID).Equals(first.Result.ID)
}

func TestMageOS_AnonymousVoteCountsOnceAndCanBeRemoved(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Anonymous voting target", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)
	voter, voterCtx := newAnonymousVoter(t, "token-a")

	for i := 0; i < 2; i++ {
		bus.MustDispatch(voterCtx, &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: voter.ID, NetworkHash: "net-1"})
	}

	post := reloadPost(voterCtx, newPost.Result.Number)
	Expect(post.VotesCount).Equals(1)
	Expect(post.HasVoted).IsTrue()

	otherVoterPost := reloadPost(demoTenantCtx, newPost.Result.Number)
	Expect(otherVoterPost.HasVoted).IsFalse()

	bus.MustDispatch(voterCtx, &cmd.MageOSRemoveAnonymousVote{Post: newPost.Result, VoterID: voter.ID})
	post = reloadPost(voterCtx, newPost.Result.Number)
	Expect(post.VotesCount).Equals(0)
	Expect(post.HasVoted).IsFalse()
}

func TestMageOS_NetworkDailyCapPerPost(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Popular in one office", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)

	tokens := []string{"t1", "t2", "t3"}
	capped := []bool{}
	for _, token := range tokens {
		voter, voterCtx := newAnonymousVoter(t, token)
		add := &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: voter.ID, NetworkHash: "office", VotesPerNetworkPerPostPerDay: 2}
		bus.MustDispatch(voterCtx, add)
		capped = append(capped, add.Capped)
	}
	Expect(capped).Equals([]bool{false, false, true})

	voter, voterCtx := newAnonymousVoter(t, "t4")
	add := &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: voter.ID, NetworkHash: "elsewhere", VotesPerNetworkPerPostPerDay: 2}
	bus.MustDispatch(voterCtx, add)
	Expect(add.Capped).IsFalse()

	Expect(reloadPost(demoTenantCtx, newPost.Result.Number).VotesCount).Equals(3)
}

func TestMageOS_MergeOnSignInDoesNotDoubleCount(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	both := &cmd.AddNewPost{Title: "Voted both ways", Description: "x"}
	anonOnly := &cmd.AddNewPost{Title: "Voted anonymously only", Description: "x"}
	bus.MustDispatch(jonSnowCtx, both, anonOnly)

	voter, voterCtx := newAnonymousVoter(t, "token-arya")
	bus.MustDispatch(voterCtx, &cmd.MageOSSetAnonymousVoterRoles{VoterID: voter.ID, Roles: []string{"developer", "agency"}})
	bus.MustDispatch(voterCtx, &cmd.MageOSAddAnonymousVote{Post: both.Result, VoterID: voter.ID})
	bus.MustDispatch(voterCtx, &cmd.MageOSAddAnonymousVote{Post: anonOnly.Result, VoterID: voter.ID})
	bus.MustDispatch(aryaStarkCtx, &cmd.AddVote{Post: both.Result, User: aryaStark})
	Expect(reloadPost(demoTenantCtx, both.Result.Number).VotesCount).Equals(2)

	merge := &cmd.MageOSMergeAnonymousVoter{VoterID: voter.ID, UserID: aryaStark.ID}
	bus.MustDispatch(aryaStarkCtx, merge)
	Expect(merge.MovedVotes).Equals(2)

	Expect(reloadPost(demoTenantCtx, both.Result.Number).VotesCount).Equals(1)
	moved := reloadPost(aryaStarkCtx, anonOnly.Result.Number)
	Expect(moved.VotesCount).Equals(1)
	Expect(moved.HasVoted).IsTrue()

	roles := &query.MageOSGetUserRoles{}
	bus.MustDispatch(aryaStarkCtx, roles)
	Expect(roles.Roles).Equals([]string{"developer", "agency"})

	again := &query.MageOSGetAnonymousVoter{TokenHash: mageos.HashToken("token-arya")}
	bus.MustDispatch(demoTenantCtx, again)
	Expect(*again.Result.MergedIntoUserID).Equals(aryaStark.ID)
}

func TestMageOS_MergeKeepsRolesTheUserAlreadyChose(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	bus.MustDispatch(aryaStarkCtx, &cmd.MageOSSetUserRoles{Roles: []string{"merchant"}})
	voter, voterCtx := newAnonymousVoter(t, "token-arya")
	bus.MustDispatch(voterCtx, &cmd.MageOSSetAnonymousVoterRoles{VoterID: voter.ID, Roles: []string{"developer"}})
	bus.MustDispatch(aryaStarkCtx, &cmd.MageOSMergeAnonymousVoter{VoterID: voter.ID, UserID: aryaStark.ID})

	roles := &query.MageOSGetUserRoles{}
	bus.MustDispatch(aryaStarkCtx, roles)
	Expect(roles.Roles).Equals([]string{"merchant"})
}

func TestMageOS_HistoricalVotesAreCountedAndReportedSeparately(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Imported from GitHub", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)
	_, err := trx.Execute(`UPDATE posts SET mageos_historical_votes = 6, mageos_historical_source = 'https://github.com/orgs/mage-os-lab/discussions/2' WHERE id = $1`, newPost.Result.ID)
	Expect(err).IsNil()
	bus.MustDispatch(jonSnowCtx, &cmd.AddVote{Post: newPost.Result, User: jonSnow})

	post := reloadPost(demoTenantCtx, newPost.Result.Number)
	Expect(post.VotesCount).Equals(7)
	Expect(post.HistoricalVotesCount).Equals(6)
	Expect(post.HistoricalSource).Equals("https://github.com/orgs/mage-os-lab/discussions/2")
}

func TestMageOS_ParticipationByRoleCountsMultiRoleVotersInEachGroup(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Role counts", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)

	bus.MustDispatch(jonSnowCtx, &cmd.MageOSSetUserRoles{Roles: []string{"developer", "agency"}})
	bus.MustDispatch(jonSnowCtx, &cmd.AddVote{Post: newPost.Result, User: jonSnow})
	voter, voterCtx := newAnonymousVoter(t, "anon-dev")
	bus.MustDispatch(voterCtx, &cmd.MageOSSetAnonymousVoterRoles{VoterID: voter.ID, Roles: []string{"developer"}})
	bus.MustDispatch(voterCtx, &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: voter.ID})
	silent, silentCtx := newAnonymousVoter(t, "anon-no-role")
	bus.MustDispatch(silentCtx, &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: silent.ID})

	q := &query.MageOSParticipationByRole{PostID: newPost.Result.ID}
	bus.MustDispatch(demoTenantCtx, q)
	Expect(q.TotalVoters).Equals(3)
	Expect(q.Result).HasLen(2)
	Expect(q.Result[0].Role).Equals("agency")
	Expect(q.Result[0].Participants).Equals(1)
	Expect(q.Result[1].Role).Equals("developer")
	Expect(q.Result[1].Participants).Equals(2)
}

func TestMageOS_VoidBurstOnlyRemovesThatNetworksAnonymousVotes(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Targeted by a script", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)
	bus.MustDispatch(jonSnowCtx, &cmd.AddVote{Post: newPost.Result, User: jonSnow})
	for _, token := range []string{"bot1", "bot2", "bot3"} {
		voter, voterCtx := newAnonymousVoter(t, token)
		bus.MustDispatch(voterCtx, &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: voter.ID, NetworkHash: "botnet"})
	}
	human, humanCtx := newAnonymousVoter(t, "human")
	bus.MustDispatch(humanCtx, &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: human.ID, NetworkHash: "home"})

	bursts := &query.MageOSListVoteBursts{Since: time.Now().Add(-time.Hour), MinVotes: 3}
	bus.MustDispatch(demoTenantCtx, bursts)
	Expect(bursts.Result).HasLen(1)
	Expect(bursts.Result[0].NetworkHash).Equals("botnet")

	void := &cmd.MageOSVoidAnonymousVotes{
		NetworkHash: "botnet",
		PostNumber:  newPost.Result.Number,
		Since:       bursts.Result[0].FirstVoteAt,
		Until:       bursts.Result[0].LastVoteAt,
	}
	bus.MustDispatch(jonSnowCtx, void)
	Expect(void.Deleted).Equals(3)
	Expect(reloadPost(demoTenantCtx, newPost.Result.Number).VotesCount).Equals(2)
}

func TestMageOS_PurgeNetworkHashes(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Old vote", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)
	voter, voterCtx := newAnonymousVoter(t, "old")
	bus.MustDispatch(voterCtx, &cmd.MageOSAddAnonymousVote{Post: newPost.Result, VoterID: voter.ID, NetworkHash: "net"})
	_, err := trx.Execute(`UPDATE mageos_anonymous_votes SET created_at = now() - INTERVAL '31 days' WHERE voter_id = $1`, voter.ID)
	Expect(err).IsNil()

	purge := &cmd.MageOSPurgeNetworkHashes{OlderThan: time.Now().Add(-mageos.NetworkHashRetention)}
	bus.MustDispatch(demoTenantCtx, purge)
	Expect(purge.Cleared).Equals(1)
	Expect(reloadPost(demoTenantCtx, newPost.Result.Number).VotesCount).Equals(1)
}

func TestMageOS_PostLinksAndGitHubState(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	newPost := &cmd.AddNewPost{Title: "Linked idea", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)

	source := &cmd.MageOSAddPostLink{Post: newPost.Result, Kind: mageos.LinkDiscussion, URL: "https://github.com/orgs/mage-os-lab/discussions/2", Repo: "mage-os-lab", Number: 2, IsSource: true, Attribution: "@octocat"}
	pr := &cmd.MageOSAddPostLink{Post: newPost.Result, Kind: mageos.LinkPullRequest, URL: "https://github.com/mage-os/mageos-magento2/pull/356", Repo: "mage-os/mageos-magento2", Number: 356}
	bus.MustDispatch(jonSnowCtx, pr, source)
	bus.MustDispatch(jonSnowCtx, &cmd.MageOSAddPostLink{Post: newPost.Result, Kind: pr.Kind, URL: pr.URL, Repo: pr.Repo, Number: pr.Number})

	list := &query.MageOSListPostLinks{PostID: newPost.Result.ID}
	bus.MustDispatch(demoTenantCtx, list)
	Expect(list.Result).HasLen(2)
	Expect(list.Result[0].IsSource).IsTrue()
	Expect(list.Result[0].Attribution).Equals("@octocat")

	apply := &cmd.MageOSApplyGitHubState{Repo: "mage-os/mageos-magento2", Number: 356, Kind: mageos.LinkPullRequest, State: "merged"}
	bus.MustDispatch(demoTenantCtx, apply)
	Expect(apply.PostIDs).Equals([]int{newPost.Result.ID})

	unrelated := &cmd.MageOSApplyGitHubState{Repo: "mage-os/mageos-magento2", Number: 356, Kind: mageos.LinkIssue, State: "closed"}
	bus.MustDispatch(demoTenantCtx, unrelated)
	Expect(unrelated.PostIDs).HasLen(0)

	bus.MustDispatch(demoTenantCtx, list)
	Expect(list.Result[1].State).Equals("merged")
	Expect(reloadPost(demoTenantCtx, newPost.Result.Number).Status.Name()).Equals("open")
}

func TestMageOS_GitHubDeliveryIsProcessedOnce(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	first := &cmd.MageOSRecordGitHubDelivery{DeliveryID: "abc-123", Event: "pull_request"}
	bus.MustDispatch(demoTenantCtx, first)
	Expect(first.Duplicate).IsFalse()

	retry := &cmd.MageOSRecordGitHubDelivery{DeliveryID: "abc-123", Event: "pull_request"}
	bus.MustDispatch(demoTenantCtx, retry)
	Expect(retry.Duplicate).IsTrue()
}

func TestMageOS_ImportHelpersAreIdempotent(t *testing.T) {
	SetupDatabaseTest(t)
	defer TeardownDatabaseTest()

	url := "https://github.com/orgs/mage-os-lab/discussions/2"
	find := &query.MageOSFindPostByLinkURL{URL: url}
	bus.MustDispatch(demoTenantCtx, find)
	Expect(find.Result).Equals(0)

	newPost := &cmd.AddNewPost{Title: "Imported discussion", Description: "x"}
	bus.MustDispatch(jonSnowCtx, newPost)
	bus.MustDispatch(jonSnowCtx, &cmd.MageOSAddPostLink{Post: newPost.Result, Kind: mageos.LinkDiscussion, URL: url, Repo: "mage-os-lab", Number: 2, IsSource: true})
	for i := 0; i < 2; i++ {
		bus.MustDispatch(jonSnowCtx, &cmd.MageOSSetHistoricalVotes{Post: newPost.Result, Votes: 6, Source: url})
	}

	bus.MustDispatch(demoTenantCtx, find)
	Expect(find.Result).Equals(newPost.Result.Number)
	post := reloadPost(demoTenantCtx, newPost.Result.Number)
	Expect(post.HistoricalVotesCount).Equals(6)
	Expect(post.VotesCount).Equals(6)
}
