package postgres

import (
	"context"

	"github.com/getfider/fider/app"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
	"github.com/getfider/fider/app/pkg/mageos"
)

const mageosReviewVolunteerSelect = `
	SELECT v.id, v.candidate_id, COALESCE(v.user_id, 0) AS user_id, COALESCE(u.name, '') AS user_name,
		COALESCE(v.github_login, '') AS github_login, v.created_at
	FROM mageos_review_volunteers v LEFT JOIN users u ON u.id = v.user_id AND u.tenant_id = v.tenant_id
	WHERE v.tenant_id = $1`

func mageosReviewSelectVolunteers(trx *dbx.Trx, tenantID int, where string, args ...any) ([]*entity.MageOSReviewVolunteer, error) {
	result := []*entity.MageOSReviewVolunteer{}
	if err := trx.Select(&result, mageosReviewVolunteerSelect+" AND "+where+" ORDER BY v.created_at, v.id", append([]any{tenantID}, args...)...); err != nil {
		return nil, errors.Wrap(err, "failed to list volunteers")
	}
	return result, nil
}

func mageosReviewVolunteerLabel(v *entity.MageOSReviewVolunteer) string {
	switch {
	case v.UserName != "" && v.GitHubLogin != "":
		return v.UserName + " (@" + v.GitHubLogin + ")"
	case v.UserName != "":
		return v.UserName
	default:
		return "@" + v.GitHubLogin
	}
}

func mageosReviewAddVolunteer(ctx context.Context, c *cmd.MageOSReviewAddVolunteer) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		login := ""
		if c.GitHubLogin != "" {
			if login = mageos.NormalizeGitHubLogin(c.GitHubLogin); login == "" {
				return errors.New("invalid GitHub username %s", c.GitHubLogin)
			}
		}
		if c.UserID <= 0 && login == "" {
			return errors.New("a Fider account or a GitHub username is required")
		}
		if c.UserID > 0 {
			exists, err := trx.Exists(`SELECT 1 FROM users WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.UserID)
			if err != nil {
				return errors.Wrap(err, "failed to check user %d", c.UserID)
			}
			if !exists {
				return app.ErrNotFound
			}
		}

		// Re-adding the same person returns the existing entry instead of failing on the unique index.
		var id int
		err := trx.Scalar(&id, `
			SELECT id FROM mageos_review_volunteers
			WHERE tenant_id = $1 AND candidate_id = $2 AND ((user_id = NULLIF($3, 0)) OR (lower(github_login) = lower(NULLIF($4, ''))))
			LIMIT 1`, tenant.ID, c.CandidateID, c.UserID, login)
		if err != nil && errors.Cause(err) != app.ErrNotFound {
			return errors.Wrap(err, "failed to check existing volunteer")
		}
		if id == 0 {
			err = trx.Scalar(&id, `
				INSERT INTO mageos_review_volunteers (tenant_id, candidate_id, user_id, github_login, created_by)
				SELECT $1, c.id, NULLIF($3, 0), NULLIF($4, ''), $5 FROM mageos_review_candidates c WHERE c.tenant_id = $1 AND c.id = $2
				RETURNING id`, tenant.ID, c.CandidateID, c.UserID, login, user.ID)
			if err != nil {
				if errors.Cause(err) == app.ErrNotFound {
					return app.ErrNotFound
				}
				return errors.Wrap(err, "failed to add volunteer")
			}
		}

		volunteers, err := mageosReviewSelectVolunteers(trx, tenant.ID, "v.id = $2", id)
		if err != nil {
			return err
		}
		c.Result = volunteers[0]
		return mageosReviewLog(trx, tenant, user, c.CandidateID, 0, "volunteer", id, "create", map[string]mageosReviewChange{
			"volunteer": {To: mageosReviewVolunteerLabel(c.Result)},
		})
	})
}

func mageosReviewRemoveVolunteer(ctx context.Context, c *cmd.MageOSReviewRemoveVolunteer) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		volunteers, err := mageosReviewSelectVolunteers(trx, tenant.ID, "v.id = $2", c.ID)
		if err != nil {
			return err
		}
		if len(volunteers) == 0 {
			return app.ErrNotFound
		}
		v := volunteers[0]
		if _, err := trx.Execute(`DELETE FROM mageos_review_volunteers WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.ID); err != nil {
			return errors.Wrap(err, "failed to remove volunteer %d", c.ID)
		}
		return mageosReviewLog(trx, tenant, user, v.CandidateID, 0, "volunteer", c.ID, "delete", map[string]mageosReviewChange{
			"volunteer": {From: mageosReviewVolunteerLabel(v)},
		})
	})
}
