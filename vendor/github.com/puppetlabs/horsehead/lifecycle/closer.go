package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/puppetlabs/horsehead/lifecycle/errors"
)

type (
	CloserRequireFunc        func() error
	CloserRequireContextFunc func(ctx context.Context) error
	CloserWhenFunc           func(ctx context.Context) error
)

type Closer struct {
	reqs []CloserRequireContextFunc

	whens  []chan error
	whenCh chan []error

	cancel  context.CancelFunc
	timeout time.Duration
	doneCh  chan struct{}
	errs    []error

	mut sync.RWMutex
}

func (c *Closer) Do(ctx context.Context) errors.Error {
	// Early check without lock.
	select {
	case <-c.Done():
		return c.Err()
	default:
	}

	c.mut.Lock()
	defer c.mut.Unlock()

	// Re-check with lock acquiried.
	select {
	case <-c.Done():
		return c.Err()
	default:
	}

	if c.timeout != 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	// Cancel the context that waiters are using.
	c.cancel()

	// Wait for all waiters to complete.
	c.errs = append(c.errs, <-c.whenCh...)

	// Close required delegates.
	for _, req := range c.reqs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = fmt.Errorf("%+v", r)
					}

					c.errs = append(c.errs, errors.NewCloserPanicError().WithCause(err))
				}
			}()

			if err := req(ctx); err != nil {
				c.errs = append(c.errs, err)
			}
		}()
	}

	// At this point we're done and shouldn't be invoked again.
	close(c.doneCh)

	return c.Err()
}

func (c *Closer) Done() <-chan struct{} {
	return c.doneCh
}

func (c *Closer) Err() errors.Error {
	select {
	case <-c.Done():
	default:
		return nil
	}

	switch len(c.errs) {
	case 0:
		return nil
	case 1:
		if err, ok := c.errs[0].(errors.Error); ok {
			return err
		}

		fallthrough
	default:
		cerr := errors.NewCloserError()

		for _, err := range c.errs {
			cerr = cerr.WithCause(err)
		}

		return cerr
	}
}

func (c *Closer) wait() {
	var errs []error

	for _, when := range c.whens {
		if err := <-when; err != nil {
			errs = append(errs, err)
		}
	}

	// This must be non-blocking.
	c.whenCh <- errs

	// Invoke completion. This will be a no-op if we're already in Do() in
	// another goroutine.
	c.Do(context.Background())
}

type CloserBuilder struct {
	reqs    []CloserRequireContextFunc
	whens   []CloserWhenFunc
	timeout time.Duration
}

func (cb *CloserBuilder) Require(fn CloserRequireFunc) *CloserBuilder {
	cb.reqs = append(cb.reqs, func(ctx context.Context) error {
		return fn()
	})
	return cb
}

func (cb *CloserBuilder) RequireContext(fn CloserRequireContextFunc) *CloserBuilder {
	cb.reqs = append(cb.reqs, fn)
	return cb
}

func (cb *CloserBuilder) When(fn CloserWhenFunc) *CloserBuilder {
	cb.whens = append(cb.whens, fn)
	return cb
}

func (cb *CloserBuilder) Timeout(d time.Duration) *CloserBuilder {
	cb.timeout = d
	return cb
}

func (cb CloserBuilder) Build() *Closer {
	ctx, cancel := context.WithCancel(context.Background())

	c := &Closer{
		reqs: append([]CloserRequireContextFunc{}, cb.reqs...),

		whens:  make([]chan error, len(cb.whens)),
		whenCh: make(chan []error, 1),

		cancel:  cancel,
		timeout: cb.timeout,
		doneCh:  make(chan struct{}),
	}

	if len(cb.whens) > 0 {
		for i, when := range cb.whens {
			c.whens[i] = make(chan error)
			go func(i int, when CloserWhenFunc) {
				var err error
				defer func() {
					c.whens[i] <- err
				}()

				// Cancel must happen before emission as the waiter process may not
				// be aligned with this iteration.
				defer cancel()

				defer func() {
					if r := recover(); r != nil {
						perr, ok := r.(error)
						if !ok {
							perr = fmt.Errorf("%+v", r)
						}

						err = errors.NewCloserPanicError().WithCause(perr)
					}
				}()

				err = when(ctx)
			}(i, when)
		}
		go c.wait()
	} else {
		// For whenever this gets picked up.
		c.whenCh <- nil
	}

	return c
}

func NewCloserBuilder() *CloserBuilder {
	return &CloserBuilder{}
}
