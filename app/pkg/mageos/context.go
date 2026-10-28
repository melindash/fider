package mageos

import (
	"context"

	"github.com/getfider/fider/app/models/entity"
)

type ctxKey struct{}

// VoterCtxKey holds the anonymous voter resolved from the voter cookie.
var VoterCtxKey = ctxKey{}

// VoterFromContext returns nil when the request has no anonymous voter.
func VoterFromContext(ctx context.Context) *entity.MageOSAnonymousVoter {
	voter, _ := ctx.Value(VoterCtxKey).(*entity.MageOSAnonymousVoter)
	return voter
}

// VoterIDFromContext returns 0 when the request has no anonymous voter.
func VoterIDFromContext(ctx context.Context) int {
	if voter := VoterFromContext(ctx); voter != nil {
		return voter.ID
	}
	return 0
}
