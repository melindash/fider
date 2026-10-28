package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/log"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/web"
)

type githubPayload struct {
	Action string `json:"action"`
	Issue  *struct {
		Number int `json:"number"`
	} `json:"issue"`
	PullRequest *struct {
		Number int  `json:"number"`
		Merged bool `json:"merged"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// IncomingGitHubWebhook records issue and pull request state on linked ideas.
// It is read-only towards GitHub, so it cannot cause update loops, and each delivery
// ID is processed once so GitHub's retries are safe.
func IncomingGitHubWebhook() web.HandlerFunc {
	return func(c *web.Context) error {
		secret := mageos.GitHubWebhookSecret()
		if secret == "" {
			return c.NotFound()
		}
		if !mageos.ValidGitHubSignature(secret, []byte(c.Request.Body), c.Request.GetHeader("X-Hub-Signature-256")) {
			log.Warn(c, "Rejected GitHub webhook with an invalid signature")
			return c.Unauthorized()
		}

		event := c.Request.GetHeader("X-GitHub-Event")
		deliveryID := c.Request.GetHeader("X-GitHub-Delivery")
		if deliveryID == "" {
			return c.BadRequest(web.Map{"message": "Missing X-GitHub-Delivery."})
		}
		if event != "issues" && event != "pull_request" {
			return c.Ok(web.Map{"outcome": "ignored event " + event})
		}

		record := &cmd.MageOSRecordGitHubDelivery{DeliveryID: deliveryID, Event: event}
		if err := bus.Dispatch(c, record); err != nil {
			return c.Failure(err)
		}
		if record.Duplicate {
			return c.Ok(web.Map{"outcome": "duplicate delivery"})
		}

		outcome, err := applyGitHubEvent(c, event, []byte(c.Request.Body))
		if err != nil {
			log.Error(c, err)
			return c.Failure(err)
		}
		_ = bus.Dispatch(c, &cmd.MageOSSetGitHubDeliveryOutcome{DeliveryID: deliveryID, Outcome: outcome})
		log.Infof(c, "GitHub @{Event} delivery @{Delivery}: @{Outcome}", dto.Props{"Event": event, "Delivery": deliveryID, "Outcome": outcome})
		return c.JSON(http.StatusOK, web.Map{"outcome": outcome})
	}
}

func applyGitHubEvent(c *web.Context, event string, body []byte) (string, error) {
	payload := githubPayload{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "invalid payload", nil
	}

	kind, number, merged := mageos.LinkIssue, 0, false
	if event == "pull_request" && payload.PullRequest != nil {
		kind, number, merged = mageos.LinkPullRequest, payload.PullRequest.Number, payload.PullRequest.Merged
	} else if payload.Issue != nil {
		number = payload.Issue.Number
	}
	state := mageos.LinkStateForEvent(event, payload.Action, merged)
	if number == 0 || state == "" {
		return "no state change for action " + payload.Action, nil
	}

	apply := &cmd.MageOSApplyGitHubState{Repo: strings.ToLower(payload.Repository.FullName), Number: number, Kind: kind, State: state}
	if err := bus.Dispatch(c, apply); err != nil {
		return "", err
	}
	if len(apply.PostIDs) == 0 {
		return "no linked ideas", nil
	}

	started := 0
	if mageos.GitHubAutoStatus() && mageos.StartsWork(kind, state) {
		for _, postID := range apply.PostIDs {
			getPost := &query.GetPostByID{PostID: postID}
			if err := bus.Dispatch(c, getPost); err != nil {
				return "", err
			}
			post := getPost.Result
			if post.Status != enum.PostPlanned {
				continue
			}
			// Status changes need an acting user. Act as the maintainer who marked the idea Planned,
			// so the response keeps their name and text; without one, leave it for a human.
			if post.Response == nil || post.Response.User == nil {
				log.Warnf(c, "Post @{PostID} is planned without a responder; not moving it to started", dto.Props{"PostID": post.ID})
				continue
			}
			asResponder := context.WithValue(c, app.UserCtxKey, post.Response.User)
			if err := bus.Dispatch(asResponder, &cmd.SetPostResponse{Post: post, Text: post.Response.Text, Status: enum.PostStarted}); err != nil {
				return "", err
			}
			started++
		}
	}
	return fmt.Sprintf("%s on %d linked ideas, %d moved to started", state, len(apply.PostIDs), started), nil
}
