package apiv1

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/mageos"
	"github.com/getfider/fider/app/pkg/web"
)

// Roadmap review API. Routes are limited to collaborators and administrators in routes_mageos.go.

func reviewID(c *web.Context, name string) (int, bool) {
	id, err := c.ParamAsInt(name)
	return id, err == nil && id > 0
}

func reviewFailure(c *web.Context, err error) error {
	if errors.Cause(err) == app.ErrNotFound {
		return c.NotFound()
	}
	return c.Failure(err)
}

// reviewCheckFields rejects unknown fields and out-of-vocabulary values with a 400 before they reach the store.
func reviewCheckFields(c *web.Context, allowed map[string]string, fields map[string]string) error {
	if len(fields) == 0 {
		return c.BadRequest(web.Map{"message": "Nothing to change."})
	}
	for field, value := range fields {
		if _, ok := allowed[field]; !ok {
			return c.BadRequest(web.Map{"message": "Unknown field " + field + "."})
		}
		if !mageos.ValidReviewValue(field, value) {
			return c.BadRequest(web.Map{"message": "Invalid value for " + field + "."})
		}
	}
	return nil
}

func reviewValidDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

// ReviewVocabulary returns the allowed values so the UI never hard-codes a second copy.
func ReviewVocabulary() web.HandlerFunc {
	return func(c *web.Context) error {
		return c.Ok(web.Map{
			"tracks":            mageos.ReviewTracks,
			"stages":            mageos.ReviewStages,
			"delivery":          mageos.ReviewDelivery,
			"efforts":           mageos.ReviewEfforts,
			"ratings":           mageos.ReviewRatings,
			"outcomes":          mageos.ReviewOutcomes,
			"escalations":       mageos.ReviewEscalations,
			"valueCriteria":     mageos.ReviewValueCriteria,
			"readinessCriteria": mageos.ReviewReadinessCriteria,
			"requiredFields":    mageos.ReviewRequiredFields,
			"roles":             mageos.ParticipantRoles,
		})
	}
}

func ReviewListCandidates() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.MageOSReviewListCandidates{IncludeArchived: c.QueryParam("archived") == "true"}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

func ReviewAddCandidate() web.HandlerFunc {
	return func(c *web.Context) error {
		input := struct {
			PostNumber int    `json:"postNumber"`
			Title      string `json:"title"`
			SourceURL  string `json:"sourceUrl"`
			Track      string `json:"track"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if input.Track != "" && !mageos.ValidReviewValue("track", input.Track) {
			return c.BadRequest(web.Map{"message": "Unknown track."})
		}
		add := &cmd.MageOSReviewAddCandidate{Title: input.Title, SourceURL: input.SourceURL, Track: input.Track}
		if input.PostNumber > 0 {
			getPost := &query.GetPostByNumber{Number: input.PostNumber}
			if err := bus.Dispatch(c, getPost); err != nil {
				return reviewFailure(c, err)
			}
			add.PostID = getPost.Result.ID
			add.Title = ""
		} else if strings.TrimSpace(input.Title) == "" {
			return c.BadRequest(web.Map{"message": "Enter a title, or pick an idea from the board."})
		}
		if err := bus.Dispatch(c, add); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"id": add.Result, "existed": add.Existed})
	}
}

// reviewParticipation returns vote counts by self-reported role for a linked idea, or nil.
func reviewParticipation(c *web.Context, candidate *entity.MageOSReviewCandidate) (web.Map, error) {
	if candidate.Idea == nil {
		return nil, nil
	}
	q := &query.MageOSParticipationByRole{PostID: candidate.Idea.ID}
	if err := bus.Dispatch(c, q); err != nil {
		return nil, err
	}
	return web.Map{
		"totalVoters":     q.TotalVoters,
		"noRole":          q.NoRole,
		"byRole":          q.Result,
		"historicalVotes": candidate.Idea.HistoricalVotes,
	}, nil
}

func ReviewGetCandidate() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		q := &query.MageOSReviewGetCandidate{ID: id}
		if err := bus.Dispatch(c, q); err != nil {
			return reviewFailure(c, err)
		}
		participation, err := reviewParticipation(c, q.Result)
		if err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{
			"candidate":     q.Result,
			"assessments":   q.Assessments,
			"actions":       q.Actions,
			"decisions":     q.Items,
			"history":       q.Events,
			"participation": participation,
		})
	}
}

func ReviewUpdateCandidate() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			Fields map[string]string `json:"fields"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if err := reviewCheckFields(c, mageos.ReviewCandidateFields, input.Fields); err != nil {
			return err
		}
		if err := bus.Dispatch(c, &cmd.MageOSReviewUpdateCandidate{ID: id, Fields: input.Fields}); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}

func ReviewSetAssessment() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			Criterion string `json:"criterion"`
			Rating    string `json:"rating"`
			Note      string `json:"note"`
		}{}
		if err := c.Bind(&input); err != nil || !mageos.ValidReviewCriterion(input.Criterion) || !slices.Contains(mageos.ReviewRatings, input.Rating) {
			return c.BadRequest(web.Map{"message": "Choose a criterion and a rating."})
		}
		set := &cmd.MageOSReviewSetAssessment{CandidateID: id, Criterion: input.Criterion, Rating: input.Rating, Note: strings.TrimSpace(input.Note)}
		if err := bus.Dispatch(c, set); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}

// ReviewCompare returns candidates side by side with every assessment and role breakdowns. No combined score.
func ReviewCompare() web.HandlerFunc {
	return func(c *web.Context) error {
		ids := []int{}
		for _, raw := range strings.Split(c.QueryParam("ids"), ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 || len(ids) > 12 {
			return c.BadRequest(web.Map{"message": "Pick between 1 and 12 candidates."})
		}
		result := []web.Map{}
		for _, id := range ids {
			q := &query.MageOSReviewGetCandidate{ID: id}
			if err := bus.Dispatch(c, q); err != nil {
				return reviewFailure(c, err)
			}
			participation, err := reviewParticipation(c, q.Result)
			if err != nil {
				return c.Failure(err)
			}
			result = append(result, web.Map{"candidate": q.Result, "assessments": q.Assessments, "participation": participation})
		}
		return c.Ok(result)
	}
}

func ReviewAddAction() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			ItemID   int    `json:"itemId"`
			Owner    string `json:"owner"`
			NextStep string `json:"nextStep"`
			ReviewOn string `json:"reviewOn"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if strings.TrimSpace(input.NextStep) == "" {
			return c.BadRequest(web.Map{"message": "Describe the next step."})
		}
		if input.ReviewOn != "" && !reviewValidDate(input.ReviewOn) {
			return c.BadRequest(web.Map{"message": "Use a valid review date."})
		}
		add := &cmd.MageOSReviewAddAction{CandidateID: id, ItemID: input.ItemID, Owner: input.Owner, NextStep: input.NextStep, ReviewOn: input.ReviewOn}
		if err := bus.Dispatch(c, add); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(add.Result)
	}
}

var reviewActionFields = map[string]string{"owner": "", "nextStep": "", "reviewOn": "", "done": ""}

func ReviewUpdateAction() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			Fields map[string]string `json:"fields"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if err := reviewCheckFields(c, reviewActionFields, input.Fields); err != nil {
			return err
		}
		if v, ok := input.Fields["reviewOn"]; ok && v != "" && !reviewValidDate(v) {
			return c.BadRequest(web.Map{"message": "Use a valid review date."})
		}
		if v, ok := input.Fields["nextStep"]; ok && strings.TrimSpace(v) == "" {
			return c.BadRequest(web.Map{"message": "Describe the next step."})
		}
		if v, ok := input.Fields["done"]; ok && v != "true" && v != "false" {
			return c.BadRequest(web.Map{"message": "done must be true or false."})
		}
		if err := bus.Dispatch(c, &cmd.MageOSReviewUpdateAction{ID: id, Fields: input.Fields}); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}

// ReviewLogPublicStatus records a public status change made from the review tool. The change itself goes
// through the upstream status API, which keeps its own permission check and notifications.
func ReviewLogPublicStatus() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			Status string `json:"status"`
			Text   string `json:"text"`
		}{}
		if err := c.Bind(&input); err != nil || input.Status == "" {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if err := bus.Dispatch(c, &cmd.MageOSReviewLogPublicStatus{CandidateID: id, Status: input.Status, Text: input.Text}); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}

func ReviewListMeetings() web.HandlerFunc {
	return func(c *web.Context) error {
		q := &query.MageOSReviewListMeetings{}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

type reviewMeetingInput struct {
	HeldOn  string `json:"heldOn"`
	Purpose string `json:"purpose"`
}

func ReviewCreateMeeting() web.HandlerFunc {
	return func(c *web.Context) error {
		input := reviewMeetingInput{}
		if err := c.Bind(&input); err != nil || !reviewValidDate(input.HeldOn) {
			return c.BadRequest(web.Map{"message": "Choose a meeting date."})
		}
		create := &cmd.MageOSReviewCreateMeeting{HeldOn: input.HeldOn, Purpose: strings.TrimSpace(input.Purpose)}
		if err := bus.Dispatch(c, create); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"id": create.Result})
	}
}

func ReviewUpdateMeeting() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := reviewMeetingInput{}
		if err := c.Bind(&input); err != nil || !reviewValidDate(input.HeldOn) {
			return c.BadRequest(web.Map{"message": "Choose a meeting date."})
		}
		if err := bus.Dispatch(c, &cmd.MageOSReviewUpdateMeeting{ID: id, HeldOn: input.HeldOn, Purpose: strings.TrimSpace(input.Purpose)}); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}

func ReviewGetMeeting() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		q := &query.MageOSReviewGetMeeting{ID: id}
		if err := bus.Dispatch(c, q); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{"meeting": q.Result, "items": q.Items, "history": q.Events})
	}
}

func ReviewAddItem() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			CandidateIDs []int `json:"candidateIds"`
		}{}
		if err := c.Bind(&input); err != nil || len(input.CandidateIDs) == 0 {
			return c.BadRequest(web.Map{"message": "Pick at least one candidate."})
		}
		for _, candidateID := range input.CandidateIDs {
			if err := bus.Dispatch(c, &cmd.MageOSReviewAddItem{MeetingID: id, CandidateID: candidateID}); err != nil {
				return reviewFailure(c, err)
			}
		}
		return c.Ok(web.Map{})
	}
}

const reviewFinalLocked = "This decision is final. Reopen it as a draft to change it."

func ReviewUpdateItem() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		input := struct {
			Fields map[string]string `json:"fields"`
		}{}
		if err := c.Bind(&input); err != nil {
			return c.BadRequest(web.Map{"message": "Invalid request."})
		}
		if err := reviewCheckFields(c, mageos.ReviewItemFields, input.Fields); err != nil {
			return err
		}
		update := &cmd.MageOSReviewUpdateItem{ItemID: id, Fields: input.Fields}
		if err := bus.Dispatch(c, update); err != nil {
			return reviewFailure(c, err)
		}
		if update.Refused {
			return c.BadRequest(web.Map{"message": reviewFinalLocked})
		}
		return c.Ok(web.Map{})
	}
}

func ReviewRemoveItem() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		remove := &cmd.MageOSReviewRemoveItem{ItemID: id}
		if err := bus.Dispatch(c, remove); err != nil {
			return reviewFailure(c, err)
		}
		if remove.Refused {
			return c.BadRequest(web.Map{"message": reviewFinalLocked})
		}
		return c.Ok(web.Map{})
	}
}

func ReviewFinalizeItem() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		finalize := &cmd.MageOSReviewFinalizeItem{ItemID: id}
		if err := bus.Dispatch(c, finalize); err != nil {
			return reviewFailure(c, err)
		}
		if len(finalize.Missing) > 0 {
			return c.BadRequest(web.Map{"message": "Choose an outcome and write the rationale before finalizing.", "missing": finalize.Missing})
		}
		return c.Ok(web.Map{})
	}
}

func ReviewReopenItem() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		if err := bus.Dispatch(c, &cmd.MageOSReviewReopenItem{ItemID: id}); err != nil {
			return reviewFailure(c, err)
		}
		return c.Ok(web.Map{})
	}
}

func ReviewMeetingSummary() web.HandlerFunc {
	return func(c *web.Context) error {
		id, ok := reviewID(c, "id")
		if !ok {
			return c.NotFound()
		}
		q := &query.MageOSReviewGetMeeting{ID: id}
		if err := bus.Dispatch(c, q); err != nil {
			return reviewFailure(c, err)
		}
		html, text := mageos.ReviewMeetingSummary(c.BaseURL(), q.Result, q.Items)
		return c.Ok(web.Map{"html": html, "text": text})
	}
}

func ReviewDashboard() web.HandlerFunc {
	return func(c *web.Context) error {
		today := c.QueryParam("today")
		if !reviewValidDate(today) {
			today = time.Now().Format("2006-01-02")
		}
		q := &query.MageOSReviewGetDashboard{Today: today}
		if err := bus.Dispatch(c, q); err != nil {
			return c.Failure(err)
		}
		return c.Ok(q.Result)
	}
}

// ReviewAddSampleData creates clearly labelled example candidates for trying the tool.
func ReviewAddSampleData() web.HandlerFunc {
	return func(c *web.Context) error {
		samples := []struct {
			title, track string
			fields       map[string]string
		}{
			{"[Sample] Faster admin product grid", "product", map[string]string{
				"problem": "Example only: the product grid is slow on large catalogs.", "beneficiaries": "Merchants with large catalogs",
				"effort": "medium", "openQuestions": "Example only."}},
			{"[Sample] First-time contributor guide", "contributor_experience", map[string]string{
				"problem": "Example only: new contributors do not know where to start.", "effort": "unknown"}},
			{"[Sample] Regional meetup starter kit", "community", map[string]string{
				"problem": "Example only: organisers rebuild the same materials each time.", "operatingNeeds": "Example only: someone keeps the kit current."}},
		}
		ids := []int{}
		for _, s := range samples {
			add := &cmd.MageOSReviewAddCandidate{Title: s.title, Track: s.track, IsSample: true}
			if err := bus.Dispatch(c, add); err != nil {
				return c.Failure(err)
			}
			if err := bus.Dispatch(c, &cmd.MageOSReviewUpdateCandidate{ID: add.Result, Fields: s.fields}); err != nil {
				return c.Failure(err)
			}
			ids = append(ids, add.Result)
		}
		return c.Ok(web.Map{"ids": ids})
	}
}

func ReviewRemoveSampleData() web.HandlerFunc {
	return func(c *web.Context) error {
		remove := &cmd.MageOSReviewRemoveSampleData{}
		if err := bus.Dispatch(c, remove); err != nil {
			return c.Failure(err)
		}
		return c.Ok(web.Map{"removed": remove.Removed})
	}
}
