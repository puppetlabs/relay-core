// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	vaultutil "github.com/puppetlabs/leg/vaultutil/pkg/vault"
	"github.com/puppetlabs/relay-core/pkg/install/op/vault"
)

func vaultConfig(cfg *vaultutil.VaultConfig) vault.Config {
	return vault.Config{
		Addr: cfg.VaultAddr.String(),
	}
}

func InitializeServices(ctx context.Context, cfg *vaultutil.VaultConfig) (services, error) {
	wire.Build(
		vaultConfig,
		vault.ProviderSet,
		wire.Struct(new(services), "*"),
	)

	return services{}, nil
}
