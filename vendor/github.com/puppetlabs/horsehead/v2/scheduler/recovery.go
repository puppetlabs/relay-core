package scheduler

import (
	"context"
	"reflect"
	"time"

	"github.com/puppetlabs/horsehead/v2/netutil"
	"github.com/puppetlabs/horsehead/v2/scheduler/errors"
)

const (
	defaultBackoffMultiplier = time.Millisecond * 5
	defaultResetRetriesAfter = time.Second * 10
)

// RecoveryDescriptorOptions contains fields that allow backoff and retry parameters
// to be set.
type RecoveryDescriptorOptions struct {
	// BackoffMultiplier is the timing multiplier between attempts using netutil.Backoff.
	//
	// Default: 5ms
	BackoffMultiplier time.Duration
	// MaxRetries is the max times the RecoveryDescriptor should attempt to run the delegate
	// descriptor during a reset retries duration. If this option is <= 0 then it means
	// retry inifinitly; however, the backoff multiplier still applies.
	//
	// Default: 0
	MaxRetries int
	// ResetRetriesAfter is the time it takes to reset the retry count when running
	// a delegate descriptor.
	//
	// Default: 10s
	ResetRetriesAfter time.Duration
}

// RecoveryDescriptor wraps a given descriptor so that it restarts if the
// descriptor itself fails. This is useful for descriptors that work off of
// external information (APIs, events, etc.).
type RecoveryDescriptor struct {
	delegate   Descriptor
	backoff    netutil.Backoff
	maxRetries int
	resetAfter time.Duration
}

var _ Descriptor = &RecoveryDescriptor{}

func (rd *RecoveryDescriptor) runOnce(ctx context.Context, pc chan<- Process) (bool, error) {
	err := rd.delegate.Run(ctx, pc)

	select {
	case <-ctx.Done():
		return false, err
	default:
	}

	if err != nil {
		log(ctx).Warn("restarting failing descriptor", "descriptor", reflect.TypeOf(rd.delegate).String(), "error", err)
	}

	return true, nil
}

// Run delegates work to another descriptor, catching any errors are restarting
// the descriptor immediately if an error occurs. It might return a max retries error.
// It only terminates when the context is done or the max retries have been exceeded.
func (rd *RecoveryDescriptor) Run(ctx context.Context, pc chan<- Process) error {
	var retries int

	for {
		start := time.Now()

		if cont, err := rd.runOnce(ctx, pc); err != nil {
			return err
		} else if !cont {
			break
		}

		if time.Now().Sub(start) >= rd.resetAfter {
			retries = 0
		}

		if rd.maxRetries > 0 && retries == rd.maxRetries {
			log(ctx).Error("max retries reached; stopping descriptor", "descriptor", reflect.TypeOf(rd.delegate).String())
			return errors.NewRecoveryDescriptorMaxRetriesReached(int64(rd.maxRetries))
		}

		retries++

		if err := rd.backoff.Backoff(ctx, retries); err != nil {
			return err
		}
	}

	return nil
}

// NewRecoveryDescriptor creates a new recovering descriptor wrapping the given
// delegate descriptor. Default backoff and retry parameters will be used.
func NewRecoveryDescriptor(delegate Descriptor) *RecoveryDescriptor {
	return NewRecoveryDescriptorWithOptions(delegate, RecoveryDescriptorOptions{})
}

// NewRecoveryDescriptorWithOptions creates a new recovering descriptor wrapping the
// given delegate descriptor. It takes RecoveryDescriptorOptions to tune backoff and retry
// parameters.
func NewRecoveryDescriptorWithOptions(delegate Descriptor, opts RecoveryDescriptorOptions) *RecoveryDescriptor {
	if opts.BackoffMultiplier == 0 {
		opts.BackoffMultiplier = defaultBackoffMultiplier
	}

	if opts.ResetRetriesAfter == 0 {
		opts.ResetRetriesAfter = defaultResetRetriesAfter
	}

	// TODO migrate to backoff's NextRun once implemented
	backoff := &netutil.ExponentialBackoff{Multiplier: opts.BackoffMultiplier}

	return &RecoveryDescriptor{
		delegate:   delegate,
		backoff:    backoff,
		maxRetries: opts.MaxRetries,
		resetAfter: opts.ResetRetriesAfter,
	}
}
