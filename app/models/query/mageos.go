package query

import (
	"time"

	"github.com/getfider/fider/app/models/entity"
)

type MageOSGetAnonymousVoter struct {
	TokenHash string

	Result *entity.MageOSAnonymousVoter
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
