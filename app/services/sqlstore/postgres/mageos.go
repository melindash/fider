package postgres

import (
	"context"

	"github.com/lib/pq"

	"github.com/getfider/fider/app"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/query"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/dbx"
	"github.com/getfider/fider/app/pkg/errors"
)

type dbMageOSVoter struct {
	ID               int            `db:"id"`
	Roles            pq.StringArray `db:"roles"`
	RolesPromptedAt  dbx.NullTime   `db:"roles_prompted_at"`
	MergedIntoUserID dbx.NullInt    `db:"merged_into_user_id"`
}

func (v dbMageOSVoter) toModel() *entity.MageOSAnonymousVoter {
	voter := &entity.MageOSAnonymousVoter{ID: v.ID, Roles: []string(v.Roles)}
	if v.RolesPromptedAt.Valid {
		voter.RolesPromptedAt = &v.RolesPromptedAt.Time
	}
	if v.MergedIntoUserID.Valid {
		mergedInto := int(v.MergedIntoUserID.Int64)
		voter.MergedIntoUserID = &mergedInto
	}
	return voter
}

func registerMageOSHandlers() {
	bus.AddHandler(mageosGetAnonymousVoter)
	bus.AddHandler(mageosEnsureAnonymousVoter)
	bus.AddHandler(mageosAddAnonymousVote)
	bus.AddHandler(mageosRemoveAnonymousVote)
	bus.AddHandler(mageosSetAnonymousVoterRoles)
	bus.AddHandler(mageosGetUserRoles)
	bus.AddHandler(mageosSetUserRoles)
	bus.AddHandler(mageosMergeAnonymousVoter)
	bus.AddHandler(mageosParticipationByRole)
	bus.AddHandler(mageosListVoteBursts)
	bus.AddHandler(mageosVoidAnonymousVotes)
	bus.AddHandler(mageosPurgeNetworkHashes)
}

const mageosVoterColumns = `id, roles, roles_prompted_at, merged_into_user_id`

func mageosGetAnonymousVoter(ctx context.Context, q *query.MageOSGetAnonymousVoter) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		voter := dbMageOSVoter{}
		err := trx.Get(&voter, `SELECT `+mageosVoterColumns+` FROM mageos_anonymous_voters WHERE tenant_id = $1 AND token_hash = $2`, tenant.ID, q.TokenHash)
		if errors.Cause(err) == app.ErrNotFound {
			q.Result = nil
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "failed to get anonymous voter")
		}
		q.Result = voter.toModel()
		return nil
	})
}

func mageosEnsureAnonymousVoter(ctx context.Context, c *cmd.MageOSEnsureAnonymousVoter) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		voter := dbMageOSVoter{}
		// xmax = 0 only for a freshly inserted row, which tells us whether this is a new voter.
		var created bool
		err := trx.Scalar(&created, `
			INSERT INTO mageos_anonymous_voters (tenant_id, token_hash) VALUES ($1, $2)
			ON CONFLICT (tenant_id, token_hash) DO UPDATE SET last_seen_at = now()
			RETURNING (xmax = 0)`, tenant.ID, c.TokenHash)
		if err != nil {
			return errors.Wrap(err, "failed to ensure anonymous voter")
		}
		if err := trx.Get(&voter, `SELECT `+mageosVoterColumns+` FROM mageos_anonymous_voters WHERE tenant_id = $1 AND token_hash = $2`, tenant.ID, c.TokenHash); err != nil {
			return errors.Wrap(err, "failed to load anonymous voter")
		}
		c.Result = voter.toModel()
		c.Created = created
		return nil
	})
}

func mageosAddAnonymousVote(ctx context.Context, c *cmd.MageOSAddAnonymousVote) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if !c.Post.CanBeVoted() {
			return nil
		}

		if c.VotesPerNetworkPerPostPerDay > 0 && c.NetworkHash != "" {
			// trx.Count counts result rows, so a COUNT(*) has to go through Scalar.
			var count int
			err := trx.Scalar(&count, `
				SELECT COUNT(*) FROM mageos_anonymous_votes
				WHERE tenant_id = $1 AND post_id = $2 AND network_hash = $3 AND created_at > now() - INTERVAL '1 day'`,
				tenant.ID, c.Post.ID, c.NetworkHash)
			if err != nil {
				return errors.Wrap(err, "failed to count network votes")
			}
			if count >= c.VotesPerNetworkPerPostPerDay {
				c.Capped = true
				return nil
			}
		}

		_, err := trx.Execute(`
			INSERT INTO mageos_anonymous_votes (tenant_id, post_id, voter_id, created_at, network_hash)
			VALUES ($1, $2, $3, now(), NULLIF($4, ''))
			ON CONFLICT DO NOTHING`, tenant.ID, c.Post.ID, c.VoterID, c.NetworkHash)
		if err != nil {
			return errors.Wrap(err, "failed to add anonymous vote")
		}
		return nil
	})
}

func mageosRemoveAnonymousVote(ctx context.Context, c *cmd.MageOSRemoveAnonymousVote) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if !c.Post.CanBeVoted() {
			return nil
		}
		_, err := trx.Execute(`DELETE FROM mageos_anonymous_votes WHERE tenant_id = $1 AND post_id = $2 AND voter_id = $3`,
			tenant.ID, c.Post.ID, c.VoterID)
		if err != nil {
			return errors.Wrap(err, "failed to remove anonymous vote")
		}
		return nil
	})
}

func mageosSetAnonymousVoterRoles(ctx context.Context, c *cmd.MageOSSetAnonymousVoterRoles) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		_, err := trx.Execute(`
			UPDATE mageos_anonymous_voters SET roles = $3, roles_prompted_at = now()
			WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.VoterID, pq.Array(c.Roles))
		if err != nil {
			return errors.Wrap(err, "failed to set anonymous voter roles")
		}
		return nil
	})
}

func mageosGetUserRoles(ctx context.Context, q *query.MageOSGetUserRoles) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if user == nil {
			return errors.New("user is required")
		}
		row := struct {
			Roles           pq.StringArray `db:"roles"`
			RolesPromptedAt dbx.NullTime   `db:"roles_prompted_at"`
		}{}
		err := trx.Get(&row, `SELECT mageos_roles AS roles, mageos_roles_prompted_at AS roles_prompted_at FROM users WHERE tenant_id = $1 AND id = $2`, tenant.ID, user.ID)
		if err != nil {
			return errors.Wrap(err, "failed to get user roles")
		}
		q.Roles = []string(row.Roles)
		if row.RolesPromptedAt.Valid {
			q.RolesPromptedAt = &row.RolesPromptedAt.Time
		}
		return nil
	})
}

func mageosSetUserRoles(ctx context.Context, c *cmd.MageOSSetUserRoles) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		if user == nil {
			return errors.New("user is required")
		}
		_, err := trx.Execute(`UPDATE users SET mageos_roles = $3, mageos_roles_prompted_at = now() WHERE tenant_id = $1 AND id = $2`,
			tenant.ID, user.ID, pq.Array(c.Roles))
		if err != nil {
			return errors.Wrap(err, "failed to set user roles")
		}
		return nil
	})
}

func mageosMergeAnonymousVoter(ctx context.Context, c *cmd.MageOSMergeAnonymousVoter) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		// The user's own vote wins on conflict, so an idea voted on both ways counts once.
		_, err := trx.Execute(`
			INSERT INTO post_votes (tenant_id, user_id, post_id, created_at)
			SELECT tenant_id, $3, post_id, created_at FROM mageos_anonymous_votes
			WHERE tenant_id = $1 AND voter_id = $2
			ON CONFLICT DO NOTHING`, tenant.ID, c.VoterID, c.UserID)
		if err != nil {
			return errors.Wrap(err, "failed to copy anonymous votes to user")
		}

		moved, err := trx.Execute(`DELETE FROM mageos_anonymous_votes WHERE tenant_id = $1 AND voter_id = $2`, tenant.ID, c.VoterID)
		if err != nil {
			return errors.Wrap(err, "failed to delete merged anonymous votes")
		}
		c.MovedVotes = int(moved)

		// Roles the user already chose while signed in take precedence over the anonymous ones.
		_, err = trx.Execute(`
			UPDATE users u SET mageos_roles = v.roles, mageos_roles_prompted_at = COALESCE(u.mageos_roles_prompted_at, v.roles_prompted_at)
			FROM mageos_anonymous_voters v
			WHERE u.tenant_id = $1 AND u.id = $3 AND v.tenant_id = $1 AND v.id = $2
			AND cardinality(u.mageos_roles) = 0`, tenant.ID, c.VoterID, c.UserID)
		if err != nil {
			return errors.Wrap(err, "failed to copy anonymous roles to user")
		}

		_, err = trx.Execute(`
			UPDATE mageos_anonymous_voters SET merged_into_user_id = $3, merged_at = now(), roles = '{}'
			WHERE tenant_id = $1 AND id = $2`, tenant.ID, c.VoterID, c.UserID)
		if err != nil {
			return errors.Wrap(err, "failed to mark anonymous voter as merged")
		}
		return nil
	})
}

func mageosParticipationByRole(ctx context.Context, q *query.MageOSParticipationByRole) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		participants := `
			WITH participants AS (
				SELECT DISTINCT 'u' || u.id AS participant, u.mageos_roles AS roles
				FROM post_votes pv
				INNER JOIN users u ON u.id = pv.user_id AND u.tenant_id = pv.tenant_id
				WHERE pv.tenant_id = $1 AND ($2 = 0 OR pv.post_id = $2)
				UNION
				SELECT DISTINCT 'a' || v.id, v.roles
				FROM mageos_anonymous_votes av
				INNER JOIN mageos_anonymous_voters v ON v.id = av.voter_id AND v.tenant_id = av.tenant_id
				WHERE av.tenant_id = $1 AND ($2 = 0 OR av.post_id = $2)
			)`

		if err := trx.Scalar(&q.TotalVoters, participants+` SELECT COUNT(*) FROM participants`, tenant.ID, q.PostID); err != nil {
			return errors.Wrap(err, "failed to count participants")
		}

		q.Result = []*entity.MageOSRoleCount{}
		err := trx.Select(&q.Result, participants+`
			SELECT role, COUNT(*) AS participants
			FROM participants, unnest(roles) AS role
			GROUP BY role ORDER BY role`, tenant.ID, q.PostID)
		if err != nil {
			return errors.Wrap(err, "failed to count participants by role")
		}
		return nil
	})
}

func mageosListVoteBursts(ctx context.Context, q *query.MageOSListVoteBursts) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		q.Result = []*entity.MageOSVoteBurst{}
		err := trx.Select(&q.Result, `
			SELECT p.number AS post_number, p.title AS post_title, av.network_hash,
				COUNT(*) AS votes, MIN(av.created_at) AS first_vote_at, MAX(av.created_at) AS last_vote_at
			FROM mageos_anonymous_votes av
			INNER JOIN posts p ON p.id = av.post_id AND p.tenant_id = av.tenant_id
			WHERE av.tenant_id = $1 AND av.created_at >= $2 AND av.network_hash IS NOT NULL
			GROUP BY p.number, p.title, av.network_hash
			HAVING COUNT(*) >= $3
			ORDER BY votes DESC`, tenant.ID, q.Since, q.MinVotes)
		if err != nil {
			return errors.Wrap(err, "failed to list anonymous vote bursts")
		}
		return nil
	})
}

func mageosVoidAnonymousVotes(ctx context.Context, c *cmd.MageOSVoidAnonymousVotes) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		deleted, err := trx.Execute(`
			DELETE FROM mageos_anonymous_votes av
			USING posts p
			WHERE p.id = av.post_id AND p.tenant_id = av.tenant_id
			AND av.tenant_id = $1 AND av.network_hash = $2 AND p.number = $3
			AND av.created_at >= $4 AND av.created_at <= $5`,
			tenant.ID, c.NetworkHash, c.PostNumber, c.Since, c.Until)
		if err != nil {
			return errors.Wrap(err, "failed to void anonymous votes")
		}
		c.Deleted = int(deleted)
		return nil
	})
}

func mageosPurgeNetworkHashes(ctx context.Context, c *cmd.MageOSPurgeNetworkHashes) error {
	return using(ctx, func(trx *dbx.Trx, tenant *entity.Tenant, user *entity.User) error {
		cleared, err := trx.Execute(`UPDATE mageos_anonymous_votes SET network_hash = NULL WHERE network_hash IS NOT NULL AND created_at < $1`, c.OlderThan)
		if err != nil {
			return errors.Wrap(err, "failed to purge network hashes")
		}
		c.Cleared = int(cleared)
		return nil
	})
}
