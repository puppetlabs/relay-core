package obj

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

type RetryError struct {
	Transient bool
	Cause     error
}

func RetryPermanent(err error) *RetryError {
	return &RetryError{
		Transient: false,
		Cause:     err,
	}
}

func RetryTransient(err error) *RetryError {
	return &RetryError{
		Transient: true,
		Cause:     err,
	}
}

func Retry(ctx context.Context, freq time.Duration, fn func() *RetryError) (err error) {
	ictx, cancel := context.WithCancel(ctx)
	defer cancel()

	wait.UntilWithContext(ictx, func(ctx context.Context) {
		rt := fn()
		err = rt.Cause

		if !rt.Transient || err == nil {
			cancel()
		}
	}, freq)

	if err == nil {
		err = ctx.Err()
	}

	return
}
