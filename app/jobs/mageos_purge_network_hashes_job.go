package jobs

import (
	"time"

	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/log"
	"github.com/getfider/fider/app/pkg/mageos"
)

// MageOSPurgeNetworkHashesJobHandler drops network hashes once they are too old to help with abuse review.
type MageOSPurgeNetworkHashesJobHandler struct {
}

func (e MageOSPurgeNetworkHashesJobHandler) Schedule() string {
	return "0 15 * * * *" // every hour at minute 15
}

func (e MageOSPurgeNetworkHashesJobHandler) Run(ctx Context) error {
	c := &cmd.MageOSPurgeNetworkHashes{OlderThan: time.Now().Add(-mageos.NetworkHashRetention)}
	if err := bus.Dispatch(ctx, c); err != nil {
		return err
	}
	log.Debugf(ctx, "@{Cleared} anonymous vote network hashes were cleared", dto.Props{"Cleared": c.Cleared})
	return nil
}
