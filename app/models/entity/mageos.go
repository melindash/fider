package entity

import "time"

// MageOSAnonymousVoter is a browser that voted or chose roles without signing in.
type MageOSAnonymousVoter struct {
	ID               int        `json:"-"`
	Roles            []string   `json:"roles"`
	RolesPromptedAt  *time.Time `json:"rolesPromptedAt"`
	MergedIntoUserID *int       `json:"-"`
}

// MageOSRoleCount is the number of distinct participants who selected a role.
type MageOSRoleCount struct {
	Role         string `json:"role" db:"role"`
	Participants int    `json:"participants" db:"participants"`
}

// MageOSVoteBurst is a cluster of anonymous votes on one post from one network.
type MageOSVoteBurst struct {
	PostNumber  int       `json:"postNumber" db:"post_number"`
	PostTitle   string    `json:"postTitle" db:"post_title"`
	NetworkHash string    `json:"networkHash" db:"network_hash"`
	Votes       int       `json:"votes" db:"votes"`
	FirstVoteAt time.Time `json:"firstVoteAt" db:"first_vote_at"`
	LastVoteAt  time.Time `json:"lastVoteAt" db:"last_vote_at"`
}
