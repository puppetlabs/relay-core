//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/puppetlabs/leg/vaultutil/pkg/model"
	vaultutil "github.com/puppetlabs/leg/vaultutil/pkg/vault"
	"github.com/puppetlabs/relay-core/pkg/install/op/kube"
	"github.com/puppetlabs/relay-core/pkg/install/op/vault"
	vaultinit "github.com/puppetlabs/relay-core/pkg/install/vault"
)

func vaultConfigMapper(vaultConfig *vaultutil.VaultConfig) vault.Config {
	return vault.Config{
		Addr: vaultConfig.VaultAddr.String(),
	}
}

func NewVaultInitializer(ctx context.Context,
	vaultConfig *vaultutil.VaultConfig, vaultCoreConfig *vaultinit.VaultCoreConfig) (*vaultinit.VaultInitializer, error) {
	panic(wire.Build(
		kube.ProviderSet,
		vaultConfigMapper,
		vault.ProviderSet,
		vaultutil.VaultInitializationManagerProviderSet,
		vaultutil.VaultSystemManagerProviderSet,
		wire.Bind(new(model.VaultInitializationManager), new(*vaultutil.VaultInitializationManager)),
		wire.Bind(new(model.VaultSystemManager), new(*vaultutil.VaultSystemManager)),
		vaultinit.ProviderSet,
	))
}
