package op

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/memory"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
)

type SecretsBackend int

const (
	SecretsBackendVault SecretsBackend = iota
)

var (
	SecretsBackendMapping = map[string]SecretsBackend{
		"vault": SecretsBackendVault,
	}
	SecretsBackendAdapters = map[SecretsBackend]func(context.Context, *config.MetadataServerConfig) (SecretsManager, errors.Error){
		SecretsBackendVault: func(ctx context.Context, cfg *config.MetadataServerConfig) (SecretsManager, errors.Error) {
			sm, err := vault.NewVaultWithKubernetesAuth(ctx, &vault.Config{
				Addr:                       cfg.VaultAddr,
				K8sServiceAccountTokenPath: cfg.K8sServiceAccountTokenPath,
				Token:                      cfg.VaultToken,
				Role:                       cfg.VaultRole,
				Bucket:                     cfg.WorkflowID,
				EngineMount:                cfg.VaultEngineMount,
				Logger:                     cfg.Logger,
			})
			if err != nil {
				return nil, err
			}

			return NewEncodingSecretManager(sm), nil
		},
	}
)

type OutputsBackend int

const (
	OutputsBackendMemory OutputsBackend = iota
	OutputsBackendConfigmap
)

var (
	OutputsBackendMapping = map[string]OutputsBackend{
		"memory":    OutputsBackendMemory,
		"configmap": OutputsBackendConfigmap,
	}
	OutputsBackendAdapters = map[OutputsBackend]func(cfg *config.MetadataServerConfig) OutputsManager{
		OutputsBackendMemory: func(_ *config.MetadataServerConfig) OutputsManager {
			return memory.New()
		},
		OutputsBackendConfigmap: func(cfg *config.MetadataServerConfig) OutputsManager {
			return configmap.New(cfg.Kubeclient, cfg.Namespace)
		},
	}
)
