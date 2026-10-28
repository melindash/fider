package middlewares

import (
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/log"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/web"
)

// MageOSVoter resolves the anonymous voter cookie. When the visitor is signed in it
// merges the anonymous votes and roles into the account instead, which is what keeps
// a vote cast before signing in from being counted twice.
func MageOSVoter() web.MiddlewareFunc {
	return func(next web.HandlerFunc) web.HandlerFunc {
		return func(c *web.Context) error {
			cookie, err := c.Request.Cookie(mageos.VoterCookieName)
			if err != nil || cookie.Value == "" {
				return next(c)
			}

			getVoter := &query.MageOSGetAnonymousVoter{TokenHash: mageos.HashToken(cookie.Value)}
			if err := bus.Dispatch(c, getVoter); err != nil {
				return c.Failure(err)
			}

			voter := getVoter.Result
			if voter == nil || voter.MergedIntoUserID != nil {
				c.RemoveCookie(mageos.VoterCookieName)
				return next(c)
			}

			if c.User() != nil {
				merge := &cmd.MageOSMergeAnonymousVoter{VoterID: voter.ID, UserID: c.User().ID}
				if err := bus.Dispatch(c, merge); err != nil {
					return c.Failure(err)
				}
				c.RemoveCookie(mageos.VoterCookieName)
				log.Infof(c, "Merged @{Votes} anonymous votes into user @{UserID}", dto.Props{
					"Votes":  merge.MovedVotes,
					"UserID": c.User().ID,
				})
				return next(c)
			}

			c.Set(mageos.VoterCtxKey, voter)
			return next(c)
		}
	}
}
