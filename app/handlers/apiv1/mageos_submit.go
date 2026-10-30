package apiv1

import (
	"strconv"
	"strings"
	"time"

	"github.com/getfider/fider/app/actions"
	"github.com/getfider/fider/app/metrics"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/env"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/validate"
	"github.com/getfider/fider/app/pkg/web"
	"github.com/getfider/fider/app/tasks"
)

// mageosValidEmail trims and lowercases an email, returning the first validation message if it is invalid.
func mageosValidEmail(c *web.Context, email string) (string, string) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", ""
	}
	if messages := validate.Email(c, email); len(messages) > 0 {
		return email, messages[0]
	}
	return email, ""
}

// CreateAnonymousPost submits an idea without an account. The idea always waits for a moderator and
// belongs to the shared anonymous author until the submitter signs in with the same email.
func CreateAnonymousPost() web.HandlerFunc {
	return func(c *web.Context) error {
		if !mageos.AnonymousPostsEnabled() || c.Tenant().IsPrivate {
			return c.Forbidden()
		}
		if c.User() != nil {
			return c.BadRequest(web.Map{"message": "Signed-in users submit ideas with /api/v1/posts."})
		}

		input := struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Tags        []string `json:"tags"`
			Email       string   `json:"email"`
			Attachments []any    `json:"attachments"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}

		action := &actions.CreateNewPost{Title: input.Title, Description: input.Description, TagSlugs: input.Tags}
		if err := action.OnPreExecute(c); err != nil {
			return c.Failure(err)
		}
		result := action.Validate(c, nil)
		if result.Err != nil {
			return c.Failure(result.Err)
		}
		email, emailError := mageosValidEmail(c, input.Email)
		if email == "" {
			result.AddFieldFailure("email", "Enter your email address so moderators can contact you about your idea.")
		} else if emailError != "" {
			result.AddFieldFailure("email", emailError)
		}
		// Uploads from visitors without an account would be an easy way to fill the blob store.
		if len(input.Attachments) > 0 {
			result.AddFieldFailure("attachments", "Sign in to attach images.")
		}
		if len(result.Errors) > 0 {
			return c.BadRequest(web.Map{"errors": result.Errors})
		}
		for _, tag := range action.Tags {
			if !tag.IsPublic {
				return c.Forbidden()
			}
		}

		limits := mageos.CurrentLimits()
		voterID, _, limited, err := mageosEnsureVoter(c)
		if err != nil {
			return c.Failure(err)
		}
		if limited ||
			!mageos.Votes.Allow("post-network:"+mageosNetworkHash(c), limits.PostsPerNetworkPerHour, time.Hour) ||
			!mageos.Votes.Allow("post-voter:"+strconv.Itoa(voterID), limits.PostsPerVoterPerHour, time.Hour) {
			return mageosTooMany(c, "posts")
		}

		author := &cmd.MageOSEnsureAnonymousAuthor{}
		if err := bus.Dispatch(c, author); err != nil {
			return c.Failure(err)
		}
		// Upstream post creation and the new-post notification read the author from the request context.
		c.SetUser(author.Result)

		newPost := &cmd.AddNewPost{Title: action.Title, Description: action.Description}
		if err := bus.Dispatch(c, newPost); err != nil {
			return c.Failure(err)
		}
		post := newPost.Result

		if err := bus.Dispatch(c,
			&cmd.MageOSRecordAnonymousPost{Post: post, VoterID: voterID, Email: email},
			&cmd.MageOSSetAnonymousVoterEmail{VoterID: voterID, Email: email},
			&cmd.MageOSAddAnonymousVote{Post: post, VoterID: voterID, NetworkHash: mageosNetworkHash(c)},
		); err != nil {
			return c.Failure(err)
		}

		if env.Config.PostCreationWithTagsEnabled {
			for _, tag := range action.Tags {
				if err := bus.Dispatch(c, &cmd.AssignTag{Tag: tag, Post: post}); err != nil {
					return c.Failure(err)
				}
			}
		}

		c.Enqueue(tasks.NotifyAboutNewPost(post))
		metrics.TotalPosts.Inc()

		return c.Ok(web.Map{
			"id":         post.ID,
			"number":     post.Number,
			"title":      post.Title,
			"slug":       post.Slug,
			"isApproved": false,
		})
	}
}

// GetAnonymousSubmission shows collaborators the contact email left with an anonymous idea.
func GetAnonymousSubmission() web.HandlerFunc {
	return func(c *web.Context) error {
		number, err := c.ParamAsInt("number")
		if err != nil {
			return c.NotFound()
		}
		getPost := &query.GetPostByNumber{Number: number}
		if err := bus.Dispatch(c, getPost); err != nil {
			return c.Failure(err)
		}
		q := &query.MageOSGetAnonymousSubmission{PostID: getPost.Result.ID}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		if q.Result == nil {
			return c.NotFound()
		}
		return c.Ok(q.Result)
	}
}
