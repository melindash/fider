package apiv1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/log"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/web"
)

func mageosTooMany(c *web.Context, reason string) error {
	return c.JSON(http.StatusTooManyRequests, web.Map{
		"reason":  reason,
		"message": "Lots of votes are coming from your network right now. Please sign in to vote, or try again later.",
	})
}

func mageosNetworkHash(c *web.Context) string {
	ip := mageos.ClientIP(c.Request.GetHeader, c.Request.RemoteAddr())
	return mageos.NetworkHash(env.Config.JWTSecret, mageos.NetworkPrefix(ip))
}

// mageosEnsureVoter returns the request's voter, creating one and setting its cookie if needed.
// limited is true when the network has created too many voters this hour.
func mageosEnsureVoter(c *web.Context) (voterID int, created bool, limited bool, err error) {
	if voterID = mageos.VoterIDFromContext(c); voterID > 0 {
		return voterID, false, false, nil
	}
	if !mageos.Votes.Allow("new-voter:"+mageosNetworkHash(c), mageos.CurrentLimits().NewVotersPerNetworkPerHour, time.Hour) {
		return 0, false, true, nil
	}
	token := mageos.NewVoterToken()
	ensure := &cmd.MageOSEnsureAnonymousVoter{TokenHash: mageos.HashToken(token)}
	if err := bus.Dispatch(c, ensure); err != nil {
		return 0, false, false, err
	}
	c.AddCookie(mageos.VoterCookieName, token, time.Now().Add(mageos.VoterCookieLifetime))
	return ensure.Result.ID, true, false, nil
}

// AddAnonymousVote records a vote from a visitor who is not signed in.
func AddAnonymousVote() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mageos.AnonymousVotingEnabled() || c.Tenant().IsPrivate {
			return c.Forbidden()
		}
		if c.User() != nil {
			return c.BadRequest(web.Map{"message": "Signed-in users vote with /api/v1/posts/:number/votes."})
		}

		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}

		limits := mageos.CurrentLimits()
		networkHash := mageosNetworkHash(c)

		voterID, isNewVoter, limited, err := mageosEnsureVoter(c)
		if err != nil {
			return c.Failure(err)
		}
		if limited {
			return mageosTooMany(c, "new_voters")
		}

		if !mageos.Votes.Allow("voter:"+strconv.Itoa(voterID), limits.VotesPerVoterPerMinute, time.Minute) ||
			!mageos.Votes.Allow("network:"+networkHash, limits.VotesPerNetworkPerHour, time.Hour) {
			return mageosTooMany(c, "rate")
		}

		add := &cmd.MageOSAddAnonymousVote{
			Post:                         getPost.Result,
			VoterID:                      voterID,
			NetworkHash:                  networkHash,
			VotesPerNetworkPerPostPerDay: limits.VotesPerNetworkPerPostPerDay,
		}
		if err := bus.Dispatch(c, add); err != nil {
			return c.Failure(err)
		}
		if add.Capped {
			log.Warnf(c, "Anonymous vote on post @{Number} refused: network daily cap reached", dto.Props{"Number": number})
			return mageosTooMany(c, "network_cap")
		}

		return c.Ok(web.Map{"isNewVoter": isNewVoter})
	}
}

// RemoveAnonymousVote lets a visitor take back their own anonymous vote.
func RemoveAnonymousVote() web.HandlerFunc {
	return func(c *web.Context) error {
		voterID := mageos.VoterIDFromContext(c)
		if voterID == 0 {
			return c.Ok(web.Map{})
		}
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		if err := bus.Dispatch(c, &cmd.MageOSRemoveAnonymousVote{Post: getPost.Result, VoterID: voterID}); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}

type mageosParticipant struct {
	Roles          []string `json:"roles"`
	RolesPrompted  bool     `json:"rolesPrompted"`
	AvailableRoles []string `json:"availableRoles"`
	IsAnonymous    bool     `json:"isAnonymous"`
	Email          string   `json:"email,omitempty"`
}

// GetParticipant returns the current visitor's self-reported roles.
func GetParticipant() web.HandlerFunc {
	return func(c *web.Context) error {
		result := mageosParticipant{Roles: []string{}, AvailableRoles: mageos.ParticipantRoles}
		if c.User() != nil {
			q := &query.MageOSGetUserRoles{}
			if err := bus.Dispatch(c, q); err != nil {
				return c.Failure(err)
			}
			result.Roles, result.RolesPrompted = q.Roles, q.RolesPromptedAt != nil
		} else {
			result.IsAnonymous = true
			if voter := mageos.VoterFromContext(c); voter != nil {
				result.Roles, result.RolesPrompted, result.Email = voter.Roles, voter.RolesPromptedAt != nil, voter.Email
			}
		}
		if result.Roles == nil {
			result.Roles = []string{}
		}
		return c.Ok(result)
	}
}

// SetParticipantRoles saves roles. An empty list records "skipped" so the prompt stays dismissed.
// Visitors without an account may also leave an email, which roles require.
func SetParticipantRoles() web.HandlerFunc {
	return func(c *web.Context) error {
		input := struct {
			Roles []string `json:"roles"`
			Email *string  `json:"email"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if input.Roles == nil {
			input.Roles = []string{}
		}
		if !mageos.ValidRoles(input.Roles) {
			return c.BadRequest(web.Map{"message": "Unknown role."})
		}

		if c.User() != nil {
			if err := bus.Dispatch(c, &cmd.MageOSSetUserRoles{Roles: input.Roles}); err != nil {
				return c.Failure(err)
			}
			return c.Ok(web.Map{})
		}

		if !mageos.AnonymousVotingEnabled() {
			return c.Forbidden()
		}

		email := ""
		if voter := mageos.VoterFromContext(c); voter != nil {
			email = voter.Email
		}
		if input.Email != nil {
			var message string
			if email, message = mageosValidEmail(c, *input.Email); message != "" {
				return c.BadRequest(web.Map{"message": message})
			}
		}
		if len(input.Roles) > 0 && email == "" {
			return c.BadRequest(web.Map{"message": "Add your email address to save roles."})
		}

		voterID, _, limited, err := mageosEnsureVoter(c)
		if err != nil {
			return c.Failure(err)
		}
		if limited {
			return mageosTooMany(c, "new_voters")
		}
		if err := bus.Dispatch(c, &cmd.MageOSSetAnonymousVoterRoles{VoterID: voterID, Roles: input.Roles}); err != nil {
			return c.Failure(err)
		}
		if input.Email != nil {
			if err := bus.Dispatch(c, &cmd.MageOSSetAnonymousVoterEmail{VoterID: voterID, Email: email}); err != nil {
				return c.Failure(err)
			}
		}
		return c.Ok(web.Map{})
	}
}

// GetParticipationByRole returns aggregate role counts for collaborators. Never per-person data.
func GetParticipationByRole() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.MageOSParticipationByRole{}
		if number, err := strconv.Atoi(c.QueryParam("post")); err == nil && number > 0 {
			getPost := &query.GetPostByNumber{Number: number}
			if err := bus.Dispatch(c, getPost); err != nil {
				return c.Failure(err)
			}
			q.PostID = getPost.Result.ID
		}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{
			"totalVoters": q.TotalVoters,
			"noRole":      q.NoRole,
			"byRole":      q.Result,
			"note":        "Roles are self-reported and multi-select; one voter can appear in several groups. Voters without a role are counted in totalVoters only.",
		})
	}
}

// ListVoteBursts shows posts receiving many anonymous votes from one network.
func ListVoteBursts() web.HandlerFunc {
	return func(c *web.Context) error {
		hours, err := strconv.Atoi(c.QueryParam("hours"))
		if err != nil || hours <= 0 {
			hours = 24
		}
		minVotes, err := strconv.Atoi(c.QueryParam("min"))
		if err != nil || minVotes <= 0 {
			minVotes = 10
		}
		q := &query.MageOSListVoteBursts{Since: time.Now().Add(-time.Duration(hours) * time.Hour), MinVotes: minVotes}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// VoidVoteBurst deletes the anonymous votes in one burst. Votes from signed-in users are untouched.
func VoidVoteBurst() web.HandlerFunc {
	return func(c *web.Context) error {
		input := struct {
			NetworkHash string    `json:"networkHash"`
			PostNumber  int       `json:"postNumber"`
			Since       time.Time `json:"since"`
			Until       time.Time `json:"until"`
		}{}
		if err := c.Bind(&input); err != nil || input.NetworkHash == "" || input.PostNumber == 0 || input.Until.Before(input.Since) {
			return c.BadRequest(web.Map{"message": "networkHash, postNumber, since and until are required."})
		}
		void := &cmd.MageOSVoidAnonymousVotes{
			NetworkHash: input.NetworkHash,
			PostNumber:  input.PostNumber,
			Since:       input.Since,
			Until:       input.Until,
		}
		if err := bus.Dispatch(c, void); err != nil {
			return c.Failure(err)
		}
		log.Warnf(c, "User @{UserID} voided @{Deleted} anonymous votes on post @{Number}", dto.Props{
			"UserID":  c.User().ID,
			"Deleted": void.Deleted,
			"Number":  input.PostNumber,
		})
		return c.Ok(web.Map{"deleted": void.Deleted})
	}
}
