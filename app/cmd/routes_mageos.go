package cmd

import (
	"github.com/getfider/fider/app/handlers/apiv1"
	"github.com/getfider/fider/app/middlewares"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/pkg/web"
)

// mageosRoutes registers the fork's anonymous voting, anonymous idea, participant role and GitHub link API.
func mageosRoutes(r *web.Engine) {
	mageosApi := r.Group()
	{
		mageosApi.Get("/api/v1/mageos/participant", apiv1.GetParticipant())
		mageosApi.Get("/api/v1/posts/:number/links", apiv1.ListPostLinks())
		mageosApi.Use(middlewares.BlockLockedTenants())
		mageosApi.Post("/api/v1/posts/:number/anonymous-vote", apiv1.AddAnonymousVote())
		mageosApi.Delete("/api/v1/posts/:number/anonymous-vote", apiv1.RemoveAnonymousVote())
		mageosApi.Put("/api/v1/mageos/participant/roles", apiv1.SetParticipantRoles())
		mageosApi.Post("/api/v1/mageos/anonymous-posts", apiv1.CreateAnonymousPost())

		mageosApi.Use(middlewares.IsAuthenticated())
		mageosApi.Use(middlewares.IsAuthorized(enum.RoleCollaborator, enum.RoleAdministrator))
		mageosApi.Get("/api/v1/mageos/participation", apiv1.GetParticipationByRole())
		mageosApi.Get("/api/v1/posts/:number/anonymous-submission", apiv1.GetAnonymousSubmission())
		mageosApi.Get("/api/v1/mageos/vote-bursts", apiv1.ListVoteBursts())
		mageosApi.Post("/api/v1/posts/:number/links", apiv1.AddPostLink())
		mageosApi.Delete("/api/v1/posts/:number/links/:id", apiv1.DeletePostLink())
		mageosApi.Get("/api/v1/mageos/links", apiv1.FindPostByLink())

		mageosApi.Use(middlewares.IsAuthorized(enum.RoleAdministrator))
		mageosApi.Post("/api/v1/mageos/vote-bursts/void", apiv1.VoidVoteBurst())
		mageosApi.Put("/api/v1/posts/:number/historical-votes", apiv1.SetHistoricalVotes())
	}
}
