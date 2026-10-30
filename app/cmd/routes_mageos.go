package cmd

import (
	"github.com/getfider/fider/app/handlers"
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

		mageosReviewRoutes(mageosApi)

		mageosApi.Use(middlewares.IsAuthorized(enum.RoleAdministrator))
		mageosApi.Post("/api/v1/mageos/vote-bursts/void", apiv1.VoidVoteBurst())
		mageosApi.Put("/api/v1/posts/:number/historical-votes", apiv1.SetHistoricalVotes())
	}
}

// mageosReviewRoutes registers the roadmap review tool. The caller has already limited the group to collaborators and administrators.
func mageosReviewRoutes(g *web.Group) {
	g.Get("/review", handlers.Page("Roadmap review", "", "MageOSReview/Review.page"))
	g.Get("/api/v1/mageos/review/vocabulary", apiv1.ReviewVocabulary())
	g.Get("/api/v1/mageos/review/dashboard", apiv1.ReviewDashboard())
	g.Get("/api/v1/mageos/review/candidates", apiv1.ReviewListCandidates())
	g.Post("/api/v1/mageos/review/candidates", apiv1.ReviewAddCandidate())
	g.Get("/api/v1/mageos/review/candidates/:id", apiv1.ReviewGetCandidate())
	g.Put("/api/v1/mageos/review/candidates/:id", apiv1.ReviewUpdateCandidate())
	g.Put("/api/v1/mageos/review/candidates/:id/assessment", apiv1.ReviewSetAssessment())
	g.Post("/api/v1/mageos/review/candidates/:id/actions", apiv1.ReviewAddAction())
	g.Post("/api/v1/mageos/review/candidates/:id/public-status-log", apiv1.ReviewLogPublicStatus())
	g.Get("/api/v1/mageos/review/compare", apiv1.ReviewCompare())
	g.Put("/api/v1/mageos/review/actions/:id", apiv1.ReviewUpdateAction())
	g.Get("/api/v1/mageos/review/meetings", apiv1.ReviewListMeetings())
	g.Post("/api/v1/mageos/review/meetings", apiv1.ReviewCreateMeeting())
	g.Get("/api/v1/mageos/review/meetings/:id", apiv1.ReviewGetMeeting())
	g.Put("/api/v1/mageos/review/meetings/:id", apiv1.ReviewUpdateMeeting())
	g.Post("/api/v1/mageos/review/meetings/:id/items", apiv1.ReviewAddItem())
	g.Get("/api/v1/mageos/review/meetings/:id/summary", apiv1.ReviewMeetingSummary())
	g.Put("/api/v1/mageos/review/items/:id", apiv1.ReviewUpdateItem())
	g.Delete("/api/v1/mageos/review/items/:id", apiv1.ReviewRemoveItem())
	g.Post("/api/v1/mageos/review/items/:id/finalize", apiv1.ReviewFinalizeItem())
	g.Post("/api/v1/mageos/review/items/:id/reopen", apiv1.ReviewReopenItem())
	g.Post("/api/v1/mageos/review/sample", apiv1.ReviewAddSampleData())
	g.Delete("/api/v1/mageos/review/sample", apiv1.ReviewRemoveSampleData())
}
