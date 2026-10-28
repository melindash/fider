package apiv1

import (
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/web"
)

// ListPostLinks returns the GitHub links of a post.
func ListPostLinks() web.HandlerFunc {
	return func(c *web.Context) error {
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		q := &query.MageOSListPostLinks{PostID: getPost.Result.ID}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// AddPostLink links a post to a GitHub discussion, issue or pull request. Collaborators only.
func AddPostLink() web.HandlerFunc {
	return func(c *web.Context) error {
		input := struct {
			URL         string `json:"url"`
			IsSource    bool   `json:"isSource"`
			Attribution string `json:"attribution"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		target, ok := mageos.ParseGitHubURL(input.URL)
		if !ok {
			return c.BadRequest(web.Map{"message": "Use a https://github.com link to a discussion, issue or pull request."})
		}
		if len(input.Attribution) > 200 {
			return c.BadRequest(web.Map{"message": "Attribution must be 200 characters or fewer."})
		}

		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		add := &cmd.MageOSAddPostLink{
			Post:        getPost.Result,
			Kind:        target.Kind,
			URL:         target.URL,
			Repo:        target.Repo,
			Number:      target.Number,
			IsSource:    input.IsSource,
			Attribution: input.Attribution,
		}
		if err := bus.Dispatch(c, add); err != nil {
			return c.Failure(err)
		}
		return c.Ok(add.Result)
	}
}

// DeletePostLink removes a link. Collaborators only.
func DeletePostLink() web.HandlerFunc {
	return func(c *web.Context) error {
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		linkID, err := c.ParamAsInt("id")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		if err := bus.Dispatch(c, &cmd.MageOSDeletePostLink{Post: getPost.Result, LinkID: linkID}); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}

// FindPostByLink returns the post already linked to a GitHub URL, so imports can skip it.
func FindPostByLink() web.HandlerFunc {
	return func(c *web.Context) error {
		target, ok := mageos.ParseGitHubURL(c.QueryParam("url"))
		if !ok {
			return c.BadRequest(web.Map{"message": "Use a https://github.com link."})
		}
		q := &query.MageOSFindPostByLinkURL{URL: target.URL}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"postNumber": q.Result})
	}
}

// SetHistoricalVotes sets the GitHub vote baseline of an imported post. Administrators only.
func SetHistoricalVotes() web.HandlerFunc {
	return func(c *web.Context) error {
		input := struct {
			Votes  int    `json:"votes"`
			Source string `json:"source"`
		}{}
		if err := c.Bind(&input); err != nil || input.Votes < 0 || input.Votes > 100000 {
			return c.BadRequest(web.Map{"message": "votes must be between 0 and 100000."})
		}
		if input.Source != "" {
			target, ok := mageos.ParseGitHubURL(input.Source)
			if !ok {
				return c.BadRequest(web.Map{"message": "source must be a https://github.com link."})
			}
			input.Source = target.URL
		}
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		if err := bus.Dispatch(c, &cmd.MageOSSetHistoricalVotes{Post: getPost.Result, Votes: input.Votes, Source: input.Source}); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{})
	}
}
