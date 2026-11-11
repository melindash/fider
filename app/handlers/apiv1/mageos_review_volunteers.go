package apiv1

import (
	"strings"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/web"
)

func ReviewAddVolunteer() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			UserID int    `json:"userId"`
			GitHub string `json:"github"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		login := ""
		if strings.TrimSpace(input.GitHub) != "" {
			if login = mageos.NormalizeGitHubLogin(input.GitHub); login == "" {
				return c.BadRequest(web.Map{"message": "That is not a valid GitHub username."})
			}
		}
		if input.UserID <= 0 && login == "" {
			return c.BadRequest(web.Map{"message": "Pick a Fider account or enter a GitHub username."})
		}
		add := &cmd.MageOSReviewAddVolunteer{CandidateID: id, UserID: input.UserID, GitHubLogin: login}
		if err := bus.Dispatch(c, add); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(add.Result)
	}
}

func ReviewRemoveVolunteer() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		if err := bus.Dispatch(c, &cmd.MageOSReviewRemoveVolunteer{ID: id}); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}
