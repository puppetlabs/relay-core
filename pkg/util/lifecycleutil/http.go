package lifecycleutil

import (
	"context"
	"net/http"
	"time"

	"github.com/puppetlabs/horsehead/v2/lifecycle"
)

const (
	// DefaultListenWaitHTTPShutdownTimeout is the time to allow connections to
	// cleanly close. 30 seconds is the GKE preemption time, so we cut it a
	// little shorter.
	DefaultListenWaitHTTPShutdownTimeout = 25 * time.Second
)

type ListenWaitHTTPOptions struct {
	ShutdownTimeout       time.Duration
	CloserWhens           []func(ctx context.Context) error
	CloserRequireContexts []func(ctx context.Context) error
}

type ListenWaitHTTPOption func(opts *ListenWaitHTTPOptions)

func WithListenWaitHTTPCloserWhen(fn func(ctx context.Context) error) ListenWaitHTTPOption {
	return func(opts *ListenWaitHTTPOptions) {
		opts.CloserWhens = append(opts.CloserWhens, fn)
	}
}

func WithListenWaitHTTPCloserRequireContext(fn func(ctx context.Context) error) ListenWaitHTTPOption {
	return func(opts *ListenWaitHTTPOptions) {
		opts.CloserRequireContexts = append(opts.CloserRequireContexts, fn)
	}
}

func WithListenWaitHTTPShutdownTimeout(timeout time.Duration) ListenWaitHTTPOption {
	return func(opts *ListenWaitHTTPOptions) {
		opts.ShutdownTimeout = timeout
	}
}

// ListenWaitHTTP will run a server and catch the context close but allow
// existing connections to clean up nicely instead of immediately exiting.
func ListenWaitHTTP(ctx context.Context, s *http.Server, opts ...ListenWaitHTTPOption) error {
	ho := &ListenWaitHTTPOptions{
		ShutdownTimeout: DefaultListenWaitHTTPShutdownTimeout,
	}
	for _, opt := range opts {
		opt(ho)
	}

	cb := lifecycle.NewCloserBuilder().
		Timeout(ho.ShutdownTimeout).
		When(func(cctx context.Context) error {
			select {
			case <-cctx.Done():
			case <-ctx.Done():
			}

			return nil
		}).
		RequireContext(func(ctx context.Context) error {
			return s.Shutdown(ctx)
		})

	for _, when := range ho.CloserWhens {
		cb.When(when)
	}

	for _, req := range ho.CloserRequireContexts {
		cb.RequireContext(req)
	}

	closer := cb.Build()

	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-closer.Done()
	return closer.Err()
}
