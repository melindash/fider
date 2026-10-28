package cmd

import (
	"time"

	"github.com/getfider/fider/app/models/entity"
)

// MageOSEnsureAnonymousVoter finds or creates the voter for a token hash.
type MageOSEnsureAnonymousVoter struct {
	TokenHash string

	Result  *entity.MageOSAnonymousVoter
	Created bool
}

// MageOSAddAnonymousVote records a vote unless the voter already voted or the
// network has reached its daily cap for the post.
type MageOSAddAnonymousVote struct {
	Post                         *entity.Post
	VoterID                      int
	NetworkHash                  string
	VotesPerNetworkPerPostPerDay int

	Capped bool
}

type MageOSRemoveAnonymousVote struct {
	Post    *entity.Post
	VoterID int
}

type MageOSSetAnonymousVoterRoles struct {
	VoterID int
	Roles   []string
}

// MageOSSetUserRoles updates the signed-in user's roles.
type MageOSSetUserRoles struct {
	Roles []string
}

// MageOSMergeAnonymousVoter moves a voter's votes and roles onto a user who just signed in.
type MageOSMergeAnonymousVoter struct {
	VoterID int
	UserID  int

	MovedVotes int
}

// MageOSVoidAnonymousVotes deletes anonymous votes from one network in a time window.
type MageOSVoidAnonymousVotes struct {
	NetworkHash string
	PostNumber  int
	Since       time.Time
	Until       time.Time

	Deleted int
}

// MageOSPurgeNetworkHashes clears network hashes older than the retention period.
type MageOSPurgeNetworkHashes struct {
	OlderThan time.Time

	Cleared int
}
