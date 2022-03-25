package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	"github.com/puppetlabs/relay-core/pkg/install/vault"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	RuleIsConnectionRefused = errmark.RuleFunc(IsConnectionRefused)
)

func IsConnectionRefused(err error) bool {
	return strings.Contains(err.Error(), "connect: connection refused")
}

func main() {
	ctx := context.Background()
	vaultConfig, vaultCoreConfig, err := vault.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	vi, err := NewVaultInitializer(ctx, vaultConfig, vaultCoreConfig)
	if err != nil {
		log.Fatal(err)
	}

	err = retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := vi.InitializeVault(ctx); err != nil {
			retryOnError := false
			errmark.If(err, RuleIsConnectionRefused, func(err error) {
				retryOnError = true
			})

			if retryOnError {
				return retry.Repeat(err)
			}

			return retry.Done(err)
		}
		return retry.Done(nil)
	}, retry.WithBackoffFactory(
		backoff.Build(
			backoff.Exponential(500*time.Microsecond, 2),
			backoff.MaxBound(30*time.Second),
			backoff.FullJitter(),
			backoff.MaxRetries(20),
			backoff.NonSliding,
		),
	))
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(0)
}
