package mainutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/puppetlabs/horsehead/v2/lifecycle"
)

type CancelableFunc func(ctx context.Context) error

func TrapAndWait(ctx context.Context, cancelables ...CancelableFunc) int {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)

	cb := lifecycle.NewCloserBuilder().
		When(func(ctx context.Context) error {
			for {
				select {
				case sig := <-sigch:
					switch sig {
					case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt:
						return nil
					}
				case <-ctx.Done():
					return nil
				}
			}
		})
	for _, fn := range cancelables {
		cb.When(lifecycle.CloserWhenFunc(fn))
	}

	closer := cb.Build()

	select {
	case <-closer.Done():
	case <-ctx.Done():
	}

	if err := closer.Do(ctx); err != nil {
		log(ctx).Error("process ended with error", "error", err)
		return 1
	}

	return 0
}
