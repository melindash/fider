package query

import (
	"time"

	"github.com/getfider/fider/app/models/entity"
)

type MageOSGetAnonymousVoter struct {
	TokenHash string

	Result *entity.MageOSAnonymousVoter
}

// MageOSGetAnonymousSubmission returns the submitter of an anonymous idea, or nil.
type MageOSGetAnonymousSubmission struct {
	PostID int

	Result *entity.MageOSAnonymousSubmission
}

// MageOSGetUserRoles returns the signed-in user's roles.
type MageOSGetUserRoles struct {
	Roles           []string
	RolesPromptedAt *time.Time
}

// MageOSParticipationByRole counts distinct voters per role. PostID 0 means the whole board.
type MageOSParticipationByRole struct {
	PostID int

	Result      []*entity.MageOSRoleCount
	TotalVoters int
}

type MageOSListVoteBursts struct {
	Since    time.Time
	MinVotes int

	Result []*entity.MageOSVoteBurst
}

type MageOSListPostLinks struct {
	PostID int

	Result []*entity.MageOSPostLink
}

// MageOSFindPostByLinkURL returns the number of a post linked to URL, or 0.
type MageOSFindPostByLinkURL struct {
	URL string

	Result int
}
