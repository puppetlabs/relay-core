package netutil

import (
	"context"
	"math"
	"time"

	"github.com/puppetlabs/horsehead/v2/netutil/errors"
)

// Maybe this will be useful at some point?
type Backoff interface {
	Backoff(context.Context, int) errors.Error
}

// TODO: Add NextRun method for use with schedulers
type ExponentialBackoff struct {
	Multiplier time.Duration
}

func (b *ExponentialBackoff) Backoff(ctx context.Context, iteration int) errors.Error {
	timeToWait := time.Duration(math.Exp2(float64(iteration))) * b.Multiplier
	return Wait(ctx, timeToWait)
}

func Wait(ctx context.Context, wait time.Duration) errors.Error {
	select {
	case <-ctx.Done():
		// timed out or context was cancelled
		return errors.NewBackoffContextCancelledError()
	case <-time.After(wait):
		return nil
	}
}
