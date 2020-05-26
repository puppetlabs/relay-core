package scheduler

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/scheduler/errors"
)

// WaitContext waits until the given lifecycle completes or the context is done,
// returning hsch_lifecycle_timeout_error in the latter case.
func WaitContext(ctx context.Context, lc StartedLifecycle) error {
	select {
	case <-lc.Done():
	case <-ctx.Done():
		return errors.NewLifecycleTimeoutError().WithCause(ctx.Err())
	}

	return nil
}

// CloseWaitContext terminates the given lifecycle and then gracefully waits for
// it to shut down.
func CloseWaitContext(ctx context.Context, lc StartedLifecycle) error {
	lc.Close()
	return WaitContext(ctx, lc)
}
