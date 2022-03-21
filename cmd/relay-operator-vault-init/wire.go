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

var VaultSystemManagerProviderSet = wire.NewSet(
	vaultutil.NewVaultSystemManager,
)

func vaultConfig(vaultConfig *vaultutil.VaultConfig) vault.Config {
	return vault.Config{
		Addr: vaultConfig.VaultAddr.String(),
	}
}

func NewVaultInitializer(ctx context.Context,
	config *vaultutil.VaultConfig, coreConfig *vaultinit.VaultCoreConfig) (*vaultinit.VaultInitializer, error) {
	panic(wire.Build(
		kube.ProviderSet,
		vaultConfig,
		vault.ProviderSet,
		VaultSystemManagerProviderSet,
		wire.Bind(new(model.VaultSystemManager), new(*vaultutil.VaultSystemManager)),
		vaultinit.ProviderSet,
	))
}
