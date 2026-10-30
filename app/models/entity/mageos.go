package entity

import "time"

// MageOSAnonymousVoter is a browser that voted or chose roles without signing in.
type MageOSAnonymousVoter struct {
	ID               int        `json:"-"`
	Roles            []string   `json:"roles"`
	RolesPromptedAt  *time.Time `json:"rolesPromptedAt"`
	Email            string     `json:"-"`
	MergedIntoUserID *int       `json:"-"`
}

// MageOSAnonymousSubmission is the contact address left with an idea submitted without an account.
type MageOSAnonymousSubmission struct {
	Email     string    `json:"email" db:"email"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	Claimed   bool      `json:"claimed" db:"claimed"`
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

// MageOSPostLink connects an idea to a GitHub discussion, issue or pull request.
type MageOSPostLink struct {
	ID           int        `json:"id" db:"id"`
	Kind         string     `json:"kind" db:"kind"`
	URL          string     `json:"url" db:"url"`
	Repo         string     `json:"repo" db:"repo"`
	Number       int        `json:"number" db:"number"`
	IsSource     bool       `json:"isSource" db:"is_source"`
	Attribution  string     `json:"attribution,omitempty" db:"attribution"`
	State        string     `json:"state,omitempty" db:"state"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty" db:"-"`
}
