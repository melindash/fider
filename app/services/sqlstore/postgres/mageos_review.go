package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/mageos"
)

func registerMageOSReviewHandlers() {
	bus.AddHandler(mageosReviewAddCandidate)
	bus.AddHandler(mageosReviewUpdateCandidate)
	bus.AddHandler(mageosReviewSetAssessment)
	bus.AddHandler(mageosReviewCreateMeeting)
	bus.AddHandler(mageosReviewUpdateMeeting)
	bus.AddHandler(mageosReviewAddItem)
	bus.AddHandler(mageosReviewRemoveItem)
	bus.AddHandler(mageosReviewUpdateItem)
	bus.AddHandler(mageosReviewFinalizeItem)
	bus.AddHandler(mageosReviewReopenItem)
	bus.AddHandler(mageosReviewAddAction)
	bus.AddHandler(mageosReviewUpdateAction)
	bus.AddHandler(mageosReviewLogPublicStatus)
	bus.AddHandler(mageosReviewRemoveSampleData)
	bus.AddHandler(mageosReviewListCandidates)
	bus.AddHandler(mageosReviewGetCandidate)
	bus.AddHandler(mageosReviewListMeetings)
	bus.AddHandler(mageosReviewGetMeeting)
	bus.AddHandler(mageosReviewGetAssessments)
	bus.AddHandler(mageosReviewGetDashboard)
}

type mageosReviewChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// mageosReviewLog writes one history entry in the caller's transaction, so a change and its record commit together.
func mageosReviewLog(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User, candidateID, meetingID int, entityName string, entityID int, action string, changes map[string]mageosReviewChange) error {
	if user == nil {
		return errors.New("a signed-in user is required for review changes")
	}
	if changes == nil {
		changes = map[string]mageosReviewChange{}
	}
	payload, err := json.Marshal(changes)
	if err != nil {
		return errors.Wrap(err, "failed to encode review changes")
	}
	_, err = trx.Execute(`
		INSERT INTO mageos_review_events (tenant_id, candidate_id, meeting_id, user_id, entity, entity_id, action, changes)
		VALUES ($1, NULLIF($2, 0), NULLIF($3, 0), $4, $5, $6, $7, $8)`,
		tenant.ID, candidateID, meetingID, user.ID, entityName, entityID, action, string(payload))
	if err != nil {
		return errors.Wrap(err, "failed to record review history")
	}
	return nil
}

// mageosReviewApply updates allowed columns of one row and returns what changed. Column names come only
// from the allowlist passed in, never from the request.
func mageosReviewApply(trx *dbx.Trx, table string, id, tenantID int, allowed map[string]string, fields map[string]string, extraSet string) (map[string]mageosReviewChange, error) {
	columns := make([]string, 0, len(fields))
	for field := range fields {
		column, ok := allowed[field]
		if !ok {
			return nil, errors.New("unknown field %s", field)
		}
		if !mageos.ValidReviewValue(field, fields[field]) {
			return nil, errors.New("invalid value for %s", field)
		}
		columns = append(columns, column)
	}
	if len(columns) == 0 {
		return map[string]mageosReviewChange{}, nil
	}

	current := map[string]string{}
	var raw []byte
	selectList := "json_build_object(" + jsonPairs(columns) + ")::text"
	if err := trx.Scalar(&raw, fmt.Sprintf("SELECT %s FROM %s WHERE id = $1 AND tenant_id = $2", selectList, table), id, tenantID); err != nil {
		return nil, errors.Wrap(err, "failed to load %s %d", table, id)
	}
	if err := json.Unmarshal(raw, &current); err != nil {
		return nil, errors.Wrap(err, "failed to decode %s %d", table, id)
	}

	changes := map[string]mageosReviewChange{}
	sets := []string{}
	args := []any{id, tenantID}
	for field, value := range fields {
		column := allowed[field]
		if current[column] == value {
			continue
		}
		changes[field] = mageosReviewChange{From: current[column], To: value}
		args = append(args, value)
		sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
	}
	if len(sets) == 0 {
		return changes, nil
	}
	if extraSet != "" {
		sets = append(sets, extraSet)
	}
	if _, err := trx.Execute(fmt.Sprintf("UPDATE %s SET %s WHERE id = $1 AND tenant_id = $2", table, strings.Join(sets, ", ")), args...); err != nil {
		return nil, errors.Wrap(err, "failed to update %s %d", table, id)
	}
	return changes, nil
}

func jsonPairs(columns []string) string {
	pairs := make([]string, 0, len(columns))
	for _, c := range columns {
		pairs = append(pairs, fmt.Sprintf("'%s', %s", c, c))
	}
	return strings.Join(pairs, ", ")
}

func mageosReviewAddCandidate(ctx context.Context, c *cmd.MageOSReviewAddCandidate) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if c.Track == "" {
			c.Track = "product"
		}
		if !mageos.ValidReviewValue("track", c.Track) {
			return errors.New("invalid track %s", c.Track)
		}
		if c.PostID > 0 {
			err := trx.Scalar(&c.Result, `SELECT id FROM mageos_review_candidates WHERE tenant_id = $1 AND post_id = $2`, tenant.ID, c.PostID)
			if err == nil {
				c.Existed = true
				return nil
			}
			if errors.Cause(err) != app.ErrNotFound {
				return errors.Wrap(err, "failed to look up candidate")
			}
		} else if strings.TrimSpace(c.Title) == "" {
			return errors.New("a title is required for a manual candidate")
		}

		err := trx.Scalar(&c.Result, `
			INSERT INTO mageos_review_candidates (tenant_id, post_id, title, source_url, track, is_sample, created_by)
			VALUES ($1, NULLIF($2, 0), $3, $4, $5, $6, $7)
			ON CONFLICT (tenant_id, post_id) WHERE post_id IS NOT NULL DO NOTHING
			RETURNING id`, tenant.ID, c.PostID, strings.TrimSpace(c.Title), strings.TrimSpace(c.SourceURL), c.Track, c.IsSample, user.ID)
		if errors.Cause(err) == app.ErrNotFound && c.PostID > 0 {
			// Another request added the same idea between the lookup and the insert.
			c.Existed = true
			return trx.Scalar(&c.Result, `SELECT id FROM mageos_review_candidates WHERE tenant_id = $1 AND post_id = $2`, tenant.ID, c.PostID)
		}
		if err != nil {
			return errors.Wrap(err, "failed to add candidate")
		}
		return mageosReviewLog(trx, tenant, user, c.Result, 0, "candidate", c.Result, "create", map[string]mageosReviewChange{
			"title": {To: c.Title}, "postId": {To: strconv.Itoa(c.PostID)},
		})
	})
}

func mageosReviewUpdateCandidate(ctx context.Context, c *cmd.MageOSReviewUpdateCandidate) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		changes, err := mageosReviewApply(trx, "mageos_review_candidates", c.ID, tenant.ID, mageos.ReviewCandidateFields, c.Fields, "updated_at = now()")
		if err != nil {
			return err
		}
		if len(changes) == 0 {
			return nil
		}
		return mageosReviewLog(trx, tenant, user, c.ID, 0, "candidate", c.ID, "update", changes)
	})
}

func mageosReviewSetAssessment(ctx context.Context, c *cmd.MageOSReviewSetAssessment) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if !mageos.ValidReviewCriterion(c.Criterion) || !slices.Contains(mageos.ReviewRatings, c.Rating) {
			return errors.New("invalid assessment %s=%s", c.Criterion, c.Rating)
		}
		previous := struct {
			Rating string `db:"rating"`
			Note   string `db:"note"`
		}{}
		err := trx.Get(&previous, `SELECT rating, note FROM mageos_review_assessments WHERE tenant_id = $1 AND candidate_id = $2 AND user_id = $3 AND criterion = $4`,
			tenant.ID, c.CandidateID, user.ID, c.Criterion)
		if err != nil && errors.Cause(err) != app.ErrNotFound {
			return errors.Wrap(err, "failed to load assessment")
		}
		if previous.Rating == c.Rating && previous.Note == c.Note {
			return nil
		}
		_, err = trx.Execute(`
			INSERT INTO mageos_review_assessments (tenant_id, candidate_id, user_id, criterion, rating, note, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, now())
			ON CONFLICT (tenant_id, candidate_id, user_id, criterion) DO UPDATE SET rating = $5, note = $6, updated_at = now()`,
			tenant.ID, c.CandidateID, user.ID, c.Criterion, c.Rating, c.Note)
		if err != nil {
			return errors.Wrap(err, "failed to save assessment")
		}
		return mageosReviewLog(trx, tenant, user, c.CandidateID, 0, "assessment", c.CandidateID, "assess", map[string]mageosReviewChange{
			c.Criterion: {From: previous.Rating, To: c.Rating}, c.Criterion + "Note": {From: previous.Note, To: c.Note},
		})
	})
}

func validReviewDate(value string) bool {
	_, err := time.Parse("2006-01-02", value)
	return err == nil
}

func mageosReviewCreateMeeting(ctx context.Context, c *cmd.MageOSReviewCreateMeeting) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if !validReviewDate(c.HeldOn) {
			return errors.New("invalid meeting date %s", c.HeldOn)
		}
		err := trx.Scalar(&c.Result, `
			INSERT INTO mageos_review_meetings (tenant_id, held_on, purpose, is_sample, created_by) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			tenant.ID, c.HeldOn, c.Purpose, c.IsSample, user.ID)
		if err != nil {
			return errors.Wrap(err, "failed to create meeting")
		}
		return mageosReviewLog(trx, tenant, user, 0, c.Result, "meeting", c.Result, "create", map[string]mageosReviewChange{
			"heldOn": {To: c.HeldOn}, "purpose": {To: c.Purpose},
		})
	})
}

func mageosReviewUpdateMeeting(ctx context.Context, c *cmd.MageOSReviewUpdateMeeting) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if !validReviewDate(c.HeldOn) {
			return errors.New("invalid meeting date %s", c.HeldOn)
		}
		changes, err := mageosReviewApply(trx, "mageos_review_meetings", c.ID, tenant.ID,
			map[string]string{"heldOn": "held_on", "purpose": "purpose"},
			map[string]string{"heldOn": c.HeldOn, "purpose": c.Purpose}, "")
		if err != nil || len(changes) == 0 {
			return err
		}
		return mageosReviewLog(trx, tenant, user, 0, c.ID, "meeting", c.ID, "update", changes)
	})
}

func mageosReviewAddItem(ctx context.Context, c *cmd.MageOSReviewAddItem) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		err := trx.Scalar(&c.Result, `
			INSERT INTO mageos_review_items (tenant_id, meeting_id, candidate_id, position)
			SELECT $1, m.id, c.id, COALESCE((SELECT MAX(position) + 1 FROM mageos_review_items WHERE meeting_id = m.id), 0)
			FROM mageos_review_meetings m, mageos_review_candidates c
			WHERE m.tenant_id = $1 AND m.id = $2 AND c.tenant_id = $1 AND c.id = $3
			ON CONFLICT (meeting_id, candidate_id) DO NOTHING
			RETURNING id`, tenant.ID, c.MeetingID, c.CandidateID)
		if errors.Cause(err) == app.ErrNotFound {
			err = trx.Scalar(&c.Result, `SELECT id FROM mageos_review_items WHERE tenant_id = $1 AND meeting_id = $2 AND candidate_id = $3`, tenant.ID, c.MeetingID, c.CandidateID)
			if err != nil {
				return errors.Wrap(err, "meeting or candidate not found")
			}
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "failed to add agenda item")
		}
		return mageosReviewLog(trx, tenant, user, c.CandidateID, c.MeetingID, "item", c.Result, "shortlist", nil)
	})
}

type mageosReviewItemRef struct {
	MeetingID   int  `db:"meeting_id"`
	CandidateID int  `db:"candidate_id"`
	IsFinal     bool `db:"is_final"`
}

func mageosReviewLoadItem(trx *dbx.Trx, tenantID, itemID int) (*mageosReviewItemRef, error) {
	ref := &mageosReviewItemRef{}
	err := trx.Get(ref, `SELECT meeting_id, candidate_id, finalized_at IS NOT NULL AS is_final FROM mageos_review_items WHERE tenant_id = $1 AND id = $2`, tenantID, itemID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load agenda item %d", itemID)
	}
	return ref, nil
}

func mageosReviewRemoveItem(ctx context.Context, c *cmd.MageOSReviewRemoveItem) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		ref, err := mageosReviewLoadItem(trx, tenant.ID, c.ItemID)
		if err != nil {
			return err
		}
		if ref.IsFinal {
			c.Refused = true
			return nil
		}
		if _, err := trx.Execute(`DELETE FROM mageos_review_items WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ItemID); err != nil {
			return errors.Wrap(err, "failed to remove agenda item")
		}
		return mageosReviewLog(trx, tenant, user, ref.CandidateID, ref.MeetingID, "item", c.ItemID, "unshortlist", nil)
	})
}

func mageosReviewUpdateItem(ctx context.Context, c *cmd.MageOSReviewUpdateItem) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		ref, err := mageosReviewLoadItem(trx, tenant.ID, c.ItemID)
		if err != nil {
			return err
		}
		if ref.IsFinal {
			c.Refused = true
			return nil
		}
		changes, err := mageosReviewApply(trx, "mageos_review_items", c.ItemID, tenant.ID, mageos.ReviewItemFields, c.Fields, "")
		if err != nil || len(changes) == 0 {
			return err
		}
		return mageosReviewLog(trx, tenant, user, ref.CandidateID, ref.MeetingID, "item", c.ItemID, "update", changes)
	})
}

// Outcomes that close the question for now move the candidate to "decided"; the others send it back to preparation.
var mageosReviewOutcomeStage = map[string]string{
	"needs_clarification": "preparing",
	"seek_comments":       "preparing",
	"investigate":         "preparing",
	"prioritize":          "decided",
	"defer":               "decided",
	"decline":             "decided",
}

func mageosReviewFinalizeItem(ctx context.Context, c *cmd.MageOSReviewFinalizeItem) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		ref, err := mageosReviewLoadItem(trx, tenant.ID, c.ItemID)
		if err != nil {
			return err
		}
		if ref.IsFinal {
			return nil
		}
		item := struct {
			Outcome   string `db:"outcome"`
			Rationale string `db:"rationale"`
		}{}
		if err := trx.Get(&item, `SELECT outcome, rationale FROM mageos_review_items WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ItemID); err != nil {
			return errors.Wrap(err, "failed to load decision")
		}
		c.Missing = []string{}
		if item.Outcome == "" {
			c.Missing = append(c.Missing, "outcome")
		}
		if strings.TrimSpace(item.Rationale) == "" {
			c.Missing = append(c.Missing, "rationale")
		}
		if len(c.Missing) > 0 {
			return nil
		}

		if _, err := trx.Execute(`UPDATE mageos_review_items SET finalized_by = $3, finalized_at = now() WHERE tenant_id = $1 AND id = $2`,
			tenant.ID, c.ItemID, user.ID); err != nil {
			return errors.Wrap(err, "failed to finalize decision")
		}
		if err := mageosReviewLog(trx, tenant, user, ref.CandidateID, ref.MeetingID, "item", c.ItemID, "finalize", map[string]mageosReviewChange{
			"outcome": {To: item.Outcome},
		}); err != nil {
			return err
		}

		stageChanges, err := mageosReviewApply(trx, "mageos_review_candidates", ref.CandidateID, tenant.ID, mageos.ReviewCandidateFields,
			map[string]string{"stage": mageosReviewOutcomeStage[item.Outcome]}, "updated_at = now()")
		if err != nil || len(stageChanges) == 0 {
			return err
		}
		return mageosReviewLog(trx, tenant, user, ref.CandidateID, ref.MeetingID, "candidate", ref.CandidateID, "update", stageChanges)
	})
}

func mageosReviewReopenItem(ctx context.Context, c *cmd.MageOSReviewReopenItem) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		ref, err := mageosReviewLoadItem(trx, tenant.ID, c.ItemID)
		if err != nil || !ref.IsFinal {
			return err
		}
		if _, err := trx.Execute(`UPDATE mageos_review_items SET finalized_by = NULL, finalized_at = NULL WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ItemID); err != nil {
			return errors.Wrap(err, "failed to reopen decision")
		}
		return mageosReviewLog(trx, tenant, user, ref.CandidateID, ref.MeetingID, "item", c.ItemID, "reopen", nil)
	})
}

func mageosReviewAddAction(ctx context.Context, c *cmd.MageOSReviewAddAction) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if strings.TrimSpace(c.NextStep) == "" {
			return errors.New("a next step is required")
		}
		if c.ReviewOn != "" && !validReviewDate(c.ReviewOn) {
			return errors.New("invalid review date %s", c.ReviewOn)
		}
		meetingID := 0
		if c.ItemID > 0 {
			ref, err := mageosReviewLoadItem(trx, tenant.ID, c.ItemID)
			if err != nil {
				return err
			}
			if ref.CandidateID != c.CandidateID {
				return errors.New("agenda item %d belongs to another candidate", c.ItemID)
			}
			meetingID = ref.MeetingID
		}
		var id int
		err := trx.Scalar(&id, `
			INSERT INTO mageos_review_actions (tenant_id, candidate_id, item_id, owner, next_step, review_on, created_by)
			SELECT $1, c.id, NULLIF($3, 0), $4, $5, NULLIF($6, '')::date, $7 FROM mageos_review_candidates c WHERE c.tenant_id = $1 AND c.id = $2
			RETURNING id`, tenant.ID, c.CandidateID, c.ItemID, strings.TrimSpace(c.Owner), strings.TrimSpace(c.NextStep), c.ReviewOn, user.ID)
		if err != nil {
			return errors.Wrap(err, "failed to add action")
		}
		if err := mageosReviewLog(trx, tenant, user, c.CandidateID, meetingID, "action", id, "create", map[string]mageosReviewChange{
			"owner": {To: c.Owner}, "nextStep": {To: c.NextStep}, "reviewOn": {To: c.ReviewOn},
		}); err != nil {
			return err
		}
		actions, err := mageosReviewSelectActions(trx, tenant.ID, "a.id = $2", id)
		if err != nil {
			return err
		}
		c.Result = actions[0]
		return nil
	})
}

func mageosReviewUpdateAction(ctx context.Context, c *cmd.MageOSReviewUpdateAction) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		current := struct {
			CandidateID int    `db:"candidate_id"`
			MeetingID   int    `db:"meeting_id"`
			Owner       string `db:"owner"`
			NextStep    string `db:"next_step"`
			ReviewOn    string `db:"review_on"`
			Done        bool   `db:"done"`
		}{}
		err := trx.Get(&current, `
			SELECT a.candidate_id, COALESCE(i.meeting_id, 0) AS meeting_id, a.owner, a.next_step,
				COALESCE(to_char(a.review_on, 'YYYY-MM-DD'), '') AS review_on, a.done_at IS NOT NULL AS done
			FROM mageos_review_actions a LEFT JOIN mageos_review_items i ON i.id = a.item_id
			WHERE a.tenant_id = $1 AND a.id = $2`, tenant.ID, c.ID)
		if err != nil {
			return errors.Wrap(err, "failed to load action %d", c.ID)
		}

		changes := map[string]mageosReviewChange{}
		for field, value := range c.Fields {
			var column, from string
			switch field {
			case "owner":
				column, from, value = "owner = $3", current.Owner, strings.TrimSpace(value)
			case "nextStep":
				if strings.TrimSpace(value) == "" {
					return errors.New("a next step is required")
				}
				column, from, value = "next_step = $3", current.NextStep, strings.TrimSpace(value)
			case "reviewOn":
				if value != "" && !validReviewDate(value) {
					return errors.New("invalid review date %s", value)
				}
				column, from = "review_on = NULLIF($3, '')::date", current.ReviewOn
			case "done":
				column, from = "done_at = CASE WHEN $3 = 'true' THEN now() END", strconv.FormatBool(current.Done)
			default:
				return errors.New("unknown field %s", field)
			}
			if from == value {
				continue
			}
			if _, err := trx.Execute(`UPDATE mageos_review_actions SET `+column+` WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ID, value); err != nil {
				return errors.Wrap(err, "failed to update action %d", c.ID)
			}
			changes[field] = mageosReviewChange{From: from, To: value}
		}
		if len(changes) == 0 {
			return nil
		}
		return mageosReviewLog(trx, tenant, user, current.CandidateID, current.MeetingID, "action", c.ID, "update", changes)
	})
}

func mageosReviewLogPublicStatus(ctx context.Context, c *cmd.MageOSReviewLogPublicStatus) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		return mageosReviewLog(trx, tenant, user, c.CandidateID, 0, "idea", c.CandidateID, "public_status", map[string]mageosReviewChange{
			"status": {To: c.Status}, "publicText": {To: c.Text},
		})
	})
}

func mageosReviewRemoveSampleData(ctx context.Context, c *cmd.MageOSReviewRemoveSampleData) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if _, err := trx.Execute(`DELETE FROM mageos_review_meetings WHERE tenant_id = $1 AND is_sample`, tenant.ID); err != nil {
			return errors.Wrap(err, "failed to remove sample meetings")
		}
		removed, err := trx.Execute(`DELETE FROM mageos_review_candidates WHERE tenant_id = $1 AND is_sample`, tenant.ID)
		if err != nil {
			return errors.Wrap(err, "failed to remove sample candidates")
		}
		c.Removed = int(removed)
		return mageosReviewLog(trx, tenant, user, 0, 0, "sample", 0, "remove", map[string]mageosReviewChange{
			"candidates": {To: strconv.Itoa(c.Removed)},
		})
	})
}

type dbMageOSReviewCandidate struct {
	ID              int            `db:"id"`
	PostID          int            `db:"post_id"`
	Title           string         `db:"title"`
	SourceURL       string         `db:"source_url"`
	Track           string         `db:"track"`
	Stage           string         `db:"stage"`
	Delivery        string         `db:"delivery"`
	Problem         string         `db:"problem"`
	Beneficiaries   string         `db:"beneficiaries"`
	Outcome         string         `db:"outcome"`
	Scope           string         `db:"scope"`
	Exclusions      string         `db:"exclusions"`
	Evidence        string         `db:"evidence"`
	Dependencies    string         `db:"dependencies"`
	Risks           string         `db:"risks"`
	Effort          string         `db:"effort"`
	Owner           string         `db:"owner"`
	Reviewer        string         `db:"reviewer"`
	OperatingNeeds  string         `db:"operating_needs"`
	OpenQuestions   string         `db:"open_questions"`
	Links           string         `db:"links"`
	IsSample        bool           `db:"is_sample"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
	IdeaNumber      int            `db:"idea_number"`
	IdeaTitle       string         `db:"idea_title"`
	IdeaSlug        string         `db:"idea_slug"`
	IdeaDescription string         `db:"idea_description"`
	IdeaStatus      int            `db:"idea_status"`
	IdeaApproved    bool           `db:"idea_approved"`
	IdeaVotes       int            `db:"idea_votes"`
	IdeaHistorical  int            `db:"idea_historical"`
	IdeaTags        pq.StringArray `db:"idea_tags"`
	LatestOutcome   string         `db:"latest_outcome"`
	LatestDecided   string         `db:"latest_decided"`
	OpenActions     int            `db:"open_actions"`
	Assessors       int            `db:"assessors"`
}

func (d *dbMageOSReviewCandidate) toModel() *entity.MageOSReviewCandidate {
	c := &entity.MageOSReviewCandidate{
		ID: d.ID, Title: d.Title, SourceURL: d.SourceURL, Track: d.Track, Stage: d.Stage, Delivery: d.Delivery,
		Problem: d.Problem, Beneficiaries: d.Beneficiaries, Outcome: d.Outcome, Scope: d.Scope, Exclusions: d.Exclusions,
		Evidence: d.Evidence, Dependencies: d.Dependencies, Risks: d.Risks, Effort: d.Effort, Owner: d.Owner,
		Reviewer: d.Reviewer, OperatingNeeds: d.OperatingNeeds, OpenQuestions: d.OpenQuestions, Links: d.Links,
		IsSample: d.IsSample, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
		LatestOutcome: d.LatestOutcome, LatestDecided: d.LatestDecided, OpenActions: d.OpenActions, Assessors: d.Assessors,
	}
	if d.PostID > 0 {
		c.Idea = &entity.MageOSReviewIdea{
			ID: d.PostID, Number: d.IdeaNumber, Title: d.IdeaTitle, Slug: d.IdeaSlug, Description: d.IdeaDescription,
			Status: enum.PostStatus(d.IdeaStatus), Votes: d.IdeaVotes, HistoricalVotes: d.IdeaHistorical,
			Tags: []string(d.IdeaTags), IsApproved: d.IdeaApproved,
		}
		if c.Title == "" {
			c.Title = d.IdeaTitle
		}
	}
	if c.Idea != nil && c.Idea.Tags == nil {
		c.Idea.Tags = []string{}
	}
	c.Missing = mageos.ReviewMissingFields(map[string]string{
		"problem": c.Problem, "beneficiaries": c.Beneficiaries, "outcome": c.Outcome, "scope": c.Scope, "evidence": c.Evidence,
		"effort": c.Effort, "owner": c.Owner, "reviewer": c.Reviewer, "operatingNeeds": c.OperatingNeeds,
	})
	return c
}

// Linked ideas are read live; the vote total matches the board (signed-in + anonymous + carried-over GitHub votes).
const mageosReviewCandidateSelect = `
	SELECT c.id, COALESCE(c.post_id, 0) AS post_id, c.title, c.source_url, c.track, c.stage, c.delivery,
		c.problem, c.beneficiaries, c.outcome, c.scope, c.exclusions, c.evidence, c.dependencies, c.risks, c.effort,
		c.owner, c.reviewer, c.operating_needs, c.open_questions, c.links, c.is_sample, c.created_at, c.updated_at,
		COALESCE(p.number, 0) AS idea_number, COALESCE(p.title, '') AS idea_title, COALESCE(p.slug, '') AS idea_slug,
		COALESCE(p.description, '') AS idea_description, COALESCE(p.status, 0) AS idea_status, COALESCE(p.is_approved, false) AS idea_approved,
		COALESCE(p.mageos_historical_votes, 0) AS idea_historical,
		COALESCE(p.mageos_historical_votes, 0)
			+ (SELECT COUNT(*) FROM post_votes v WHERE v.tenant_id = c.tenant_id AND v.post_id = c.post_id)
			+ (SELECT COUNT(*) FROM mageos_anonymous_votes v WHERE v.tenant_id = c.tenant_id AND v.post_id = c.post_id) AS idea_votes,
		COALESCE((SELECT ARRAY_AGG(t.name ORDER BY t.name) FROM post_tags pt JOIN tags t ON t.id = pt.tag_id AND t.tenant_id = pt.tenant_id
			WHERE pt.tenant_id = c.tenant_id AND pt.post_id = c.post_id), '{}') AS idea_tags,
		COALESCE(latest.outcome, '') AS latest_outcome, COALESCE(to_char(latest.finalized_at, 'YYYY-MM-DD'), '') AS latest_decided,
		(SELECT COUNT(*) FROM mageos_review_actions a WHERE a.candidate_id = c.id AND a.done_at IS NULL) AS open_actions,
		(SELECT COUNT(DISTINCT s.user_id) FROM mageos_review_assessments s WHERE s.candidate_id = c.id) AS assessors
	FROM mageos_review_candidates c
	LEFT JOIN posts p ON p.id = c.post_id AND p.tenant_id = c.tenant_id
	LEFT JOIN LATERAL (
		SELECT i.outcome, i.finalized_at FROM mageos_review_items i
		WHERE i.candidate_id = c.id AND i.finalized_at IS NOT NULL ORDER BY i.finalized_at DESC LIMIT 1
	) latest ON true
	WHERE c.tenant_id = $1`

func mageosReviewSelectCandidates(trx *dbx.Trx, tenantID int, where string, args ...any) ([]*entity.MageOSReviewCandidate, error) {
	rows := []*dbMageOSReviewCandidate{}
	sql := mageosReviewCandidateSelect
	if where != "" {
		sql += " AND " + where
	}
	if err := trx.Select(&rows, sql+" ORDER BY c.updated_at DESC", append([]any{tenantID}, args...)...); err != nil {
		return nil, errors.Wrap(err, "failed to list review candidates")
	}
	result := make([]*entity.MageOSReviewCandidate, len(rows))
	for i, row := range rows {
		result[i] = row.toModel()
	}
	return result, nil
}

const mageosReviewActionSelect = `
	SELECT a.id, a.candidate_id, COALESCE(NULLIF(c.title, ''), p.title, '') AS candidate_title, c.is_sample,
		COALESCE(a.item_id, 0) AS item_id, a.owner, a.next_step, COALESCE(to_char(a.review_on, 'YYYY-MM-DD'), '') AS review_on,
		a.done_at IS NOT NULL AS is_done, a.created_at
	FROM mageos_review_actions a
	JOIN mageos_review_candidates c ON c.id = a.candidate_id AND c.tenant_id = a.tenant_id
	LEFT JOIN posts p ON p.id = c.post_id AND p.tenant_id = c.tenant_id
	WHERE a.tenant_id = $1`

func mageosReviewSelectActions(trx *dbx.Trx, tenantID int, where string, args ...any) ([]*entity.MageOSReviewAction, error) {
	result := []*entity.MageOSReviewAction{}
	err := trx.Select(&result, mageosReviewActionSelect+" AND "+where+" ORDER BY a.done_at IS NOT NULL, a.review_on NULLS LAST, a.id",
		append([]any{tenantID}, args...)...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list review actions")
	}
	return result, nil
}

const mageosReviewItemSelect = `
	SELECT i.id, i.meeting_id, to_char(m.held_on, 'YYYY-MM-DD') AS meeting_held_on, i.candidate_id, i.position,
		i.decision_requested, i.notes, i.outcome, i.rationale, i.objections, i.unresolved, i.escalation,
		COALESCE(u.name, '') AS finalized_by_name, COALESCE(to_char(i.finalized_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS finalized_at
	FROM mageos_review_items i
	JOIN mageos_review_meetings m ON m.id = i.meeting_id AND m.tenant_id = i.tenant_id
	LEFT JOIN users u ON u.id = i.finalized_by
	WHERE i.tenant_id = $1`

const mageosReviewEventSelect = `
	SELECT e.id, COALESCE(e.candidate_id, 0) AS candidate_id, COALESCE(e.meeting_id, 0) AS meeting_id, u.name AS user_name,
		e.entity, e.entity_id, e.action, convert_to(e.changes::text, 'UTF8') AS changes, e.created_at
	FROM mageos_review_events e JOIN users u ON u.id = e.user_id
	WHERE e.tenant_id = $1`

func mageosReviewSelectAssessments(trx *dbx.Trx, tenantID int, candidateIDs []int) ([]*entity.MageOSReviewAssessment, error) {
	result := []*entity.MageOSReviewAssessment{}
	err := trx.Select(&result, `
		SELECT s.candidate_id, s.user_id, u.name AS user_name, s.criterion, s.rating, s.note, s.updated_at
		FROM mageos_review_assessments s JOIN users u ON u.id = s.user_id
		WHERE s.tenant_id = $1 AND s.candidate_id = ANY($2) ORDER BY u.name, s.criterion`, tenantID, pq.Array(candidateIDs))
	if err != nil {
		return nil, errors.Wrap(err, "failed to list assessments")
	}
	return result, nil
}

func mageosReviewListCandidates(ctx context.Context, q *query.MageOSReviewListCandidates) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		where := "c.stage <> 'archived'"
		if q.IncludeArchived {
			where = ""
		}
		var err error
		q.Result, err = mageosReviewSelectCandidates(trx, tenant.ID, where)
		return err
	})
}

func mageosReviewGetCandidate(ctx context.Context, q *query.MageOSReviewGetCandidate) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		candidates, err := mageosReviewSelectCandidates(trx, tenant.ID, "c.id = $2", q.ID)
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return app.ErrNotFound
		}
		q.Result = candidates[0]
		if q.Assessments, err = mageosReviewSelectAssessments(trx, tenant.ID, []int{q.ID}); err != nil {
			return err
		}
		if q.Actions, err = mageosReviewSelectActions(trx, tenant.ID, "a.candidate_id = $2", q.ID); err != nil {
			return err
		}
		q.Items = []*entity.MageOSReviewItem{}
		if err := trx.Select(&q.Items, mageosReviewItemSelect+" AND i.candidate_id = $2 ORDER BY m.held_on DESC", tenant.ID, q.ID); err != nil {
			return errors.Wrap(err, "failed to list candidate decisions")
		}
		q.Events = []*entity.MageOSReviewEvent{}
		if err := trx.Select(&q.Events, mageosReviewEventSelect+" AND e.candidate_id = $2 ORDER BY e.created_at DESC, e.id DESC", tenant.ID, q.ID); err != nil {
			return errors.Wrap(err, "failed to list candidate history")
		}
		return nil
	})
}

const mageosReviewMeetingSelect = `
	SELECT m.id, to_char(m.held_on, 'YYYY-MM-DD') AS held_on, m.purpose, m.is_sample,
		(SELECT COUNT(*) FROM mageos_review_items i WHERE i.meeting_id = m.id) AS item_count,
		(SELECT COUNT(*) FROM mageos_review_items i WHERE i.meeting_id = m.id AND i.finalized_at IS NOT NULL) AS final_count
	FROM mageos_review_meetings m WHERE m.tenant_id = $1`

func mageosReviewListMeetings(ctx context.Context, q *query.MageOSReviewListMeetings) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		q.Result = []*entity.MageOSReviewMeeting{}
		if err := trx.Select(&q.Result, mageosReviewMeetingSelect+" ORDER BY m.held_on DESC, m.id DESC", tenant.ID); err != nil {
			return errors.Wrap(err, "failed to list meetings")
		}
		return nil
	})
}

func mageosReviewGetMeeting(ctx context.Context, q *query.MageOSReviewGetMeeting) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		meetings := []*entity.MageOSReviewMeeting{}
		if err := trx.Select(&meetings, mageosReviewMeetingSelect+" AND m.id = $2", tenant.ID, q.ID); err != nil {
			return errors.Wrap(err, "failed to load meeting")
		}
		if len(meetings) == 0 {
			return app.ErrNotFound
		}
		q.Result = meetings[0]

		q.Items = []*entity.MageOSReviewItem{}
		if err := trx.Select(&q.Items, mageosReviewItemSelect+" AND i.meeting_id = $2 ORDER BY i.position, i.id", tenant.ID, q.ID); err != nil {
			return errors.Wrap(err, "failed to list agenda")
		}
		ids := make([]int, len(q.Items))
		for i, item := range q.Items {
			ids[i] = item.CandidateID
		}
		candidates, err := mageosReviewSelectCandidates(trx, tenant.ID, "c.id = ANY($2)", pq.Array(ids))
		if err != nil {
			return err
		}
		assessments, err := mageosReviewSelectAssessments(trx, tenant.ID, ids)
		if err != nil {
			return err
		}
		actions, err := mageosReviewSelectActions(trx, tenant.ID, "a.candidate_id = ANY($2)", pq.Array(ids))
		if err != nil {
			return err
		}
		for _, item := range q.Items {
			item.Assessments = []*entity.MageOSReviewAssessment{}
			item.Actions = []*entity.MageOSReviewAction{}
			for _, c := range candidates {
				if c.ID == item.CandidateID {
					item.Candidate = c
				}
			}
			for _, a := range assessments {
				if a.CandidateID == item.CandidateID {
					item.Assessments = append(item.Assessments, a)
				}
			}
			for _, a := range actions {
				if a.CandidateID == item.CandidateID && (!a.IsDone || a.ItemID == item.ID) {
					item.Actions = append(item.Actions, a)
				}
			}
		}

		q.Events = []*entity.MageOSReviewEvent{}
		if err := trx.Select(&q.Events, mageosReviewEventSelect+" AND e.meeting_id = $2 ORDER BY e.created_at DESC, e.id DESC", tenant.ID, q.ID); err != nil {
			return errors.Wrap(err, "failed to list meeting history")
		}
		return nil
	})
}

func mageosReviewGetAssessments(ctx context.Context, q *query.MageOSReviewGetAssessments) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		var err error
		q.Result, err = mageosReviewSelectAssessments(trx, tenant.ID, q.CandidateIDs)
		return err
	})
}

func mageosReviewGetDashboard(ctx context.Context, q *query.MageOSReviewGetDashboard) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if !validReviewDate(q.Today) {
			return errors.New("invalid date %s", q.Today)
		}
		d := &entity.MageOSReviewDashboard{MissingInfo: []*entity.MageOSReviewCandidate{}}
		open, err := mageosReviewSelectCandidates(trx, tenant.ID, "c.stage IN ('new', 'preparing', 'ready')")
		if err != nil {
			return err
		}
		for _, c := range open {
			if len(c.Missing) > 0 {
				d.MissingInfo = append(d.MissingInfo, c)
			}
		}
		if d.UnassignedActions, err = mageosReviewSelectActions(trx, tenant.ID, "a.done_at IS NULL AND a.owner = ''"); err != nil {
			return err
		}
		if d.UpcomingReviews, err = mageosReviewSelectActions(trx, tenant.ID,
			"a.done_at IS NULL AND a.review_on >= $2::date AND a.review_on <= $2::date + 14", q.Today); err != nil {
			return err
		}
		if d.Overdue, err = mageosReviewSelectActions(trx, tenant.ID, "a.done_at IS NULL AND a.review_on < $2::date", q.Today); err != nil {
			return err
		}
		d.UpcomingMeetings = []*entity.MageOSReviewMeeting{}
		if err := trx.Select(&d.UpcomingMeetings, mageosReviewMeetingSelect+" AND m.held_on >= $2::date ORDER BY m.held_on", tenant.ID, q.Today); err != nil {
			return errors.Wrap(err, "failed to list upcoming meetings")
		}
		q.Result = d
		return nil
	})
}
