package vault

import (
	"context"
	"fmt"

	"github.com/google/wire"
	"github.com/puppetlabs/leg/vaultutil/pkg/model"
	vaultutil "github.com/puppetlabs/leg/vaultutil/pkg/vault"
)

var ProviderSet = wire.NewSet(
	NewVaultInitializer,
)

type VaultInitializer struct {
	vaultConfig                *vaultutil.VaultConfig
	vaultCoreConfig            *VaultCoreConfig
	vaultInitializationManager model.VaultInitializationManager
	vaultSystemManager         model.VaultSystemManager
}

func (vi *VaultInitializer) InitializeVault(ctx context.Context) error {
	secretEngines := []*model.VaultSecretEngine{
		{
			Path: vi.vaultCoreConfig.TenantPath,
			Type: model.VaultSecretEngineTypeKVV2.String(),
		},
		{
			Path: vi.vaultCoreConfig.TransitPath,
			Type: model.VaultSecretEngineTypeTransit.String(),
		},
	}

	if vi.vaultCoreConfig.LogServicePath != "" {
		secretEngines = append(secretEngines, &model.VaultSecretEngine{
			Path: vi.vaultCoreConfig.LogServicePath,
			Type: model.VaultSecretEngineTypeKVV2.String(),
		})
	}

	if err := vi.vaultInitializationManager.InitializeVault(ctx,
		&model.VaultInitializationData{
			SecretEngines: secretEngines,
		}); err != nil {
		return err
	}

	if err := vi.vaultSystemManager.EnableKubernetesAuth(); err != nil {
		return err
	}

	if err := vi.vaultSystemManager.ConfigureKubernetesAuth(ctx); err != nil {
		return err
	}

	if err := vi.vaultSystemManager.EnableJWTAuth(); err != nil {
		return err
	}

	if err := vi.vaultSystemManager.ConfigureJWTAuth(ctx); err != nil {
		return err
	}

	policyGen := &vaultPolicyGenerator{
		LogServicePath: vi.vaultCoreConfig.LogServicePath,
		TenantPath:     vi.vaultCoreConfig.TenantPath,
		TransitKey:     vi.vaultCoreConfig.TransitKey,
		TransitPath:    vi.vaultCoreConfig.TransitPath,
	}

	policies := []*model.VaultPolicy{
		{
			Name:  vi.vaultCoreConfig.MetadataAPIVaultAgentRole,
			Rules: string(policyGen.metadataAPIPolicy()),
		},
		{
			Name:  vi.vaultCoreConfig.OperatorVaultAgentRole,
			Rules: string(policyGen.operatorPolicy()),
		},
	}

	if vi.vaultCoreConfig.LogServiceVaultAgentRole != "" {
		policies = append(policies, &model.VaultPolicy{
			Name:  vi.vaultCoreConfig.LogServiceVaultAgentRole,
			Rules: string(policyGen.logServicePolicy()),
		})
	}

	auth, err := vi.vaultSystemManager.GetAuthMethod(fmt.Sprintf("%s/", vi.vaultConfig.JWTMount))
	if err != nil {
		return err
	}

	if auth != nil {
		policyGen.AuthJWTAccessor = auth.Accessor

		// TODO Add installer configuration for `metadata-api-tenant` name
		policies = append(policies, &model.VaultPolicy{
			Name:  "metadata-api-tenant",
			Rules: string(policyGen.metadataAPITenantPolicy()),
		})
	}

	// TODO Do not hardcode the service account names
	kubernetesRoles := []*model.VaultKubernetesRole{
		{
			Name:                          vi.vaultCoreConfig.MetadataAPIVaultAgentRole,
			BoundServiceAccountNames:      []string{"relay-core-v1-metadata-api-vault-agent"},
			BoundServiceAccountNamespaces: []string{vi.vaultConfig.Namespace},
			Policies:                      []string{vi.vaultCoreConfig.MetadataAPIVaultAgentRole},
			TTL:                           "24h",
		},
		{
			Name:                          vi.vaultCoreConfig.OperatorVaultAgentRole,
			BoundServiceAccountNames:      []string{"relay-core-v1-operator-vault-agent"},
			BoundServiceAccountNamespaces: []string{vi.vaultConfig.Namespace},
			Policies:                      []string{vi.vaultCoreConfig.OperatorVaultAgentRole},
			TTL:                           "24h",
		},
	}

	if vi.vaultCoreConfig.LogServiceVaultAgentRole != "" {
		kubernetesRoles = append(kubernetesRoles, &model.VaultKubernetesRole{
			Name:                          vi.vaultCoreConfig.LogServiceVaultAgentRole,
			BoundServiceAccountNames:      []string{"relay-core-v1-log-service-vault-agent"},
			BoundServiceAccountNamespaces: []string{vi.vaultConfig.Namespace},
			Policies:                      []string{vi.vaultCoreConfig.LogServiceVaultAgentRole},
			TTL:                           "24h",
		})
	}

	// TODO Add installer configuration for `metadata-api-tenant` name
	jwtRoles := []*model.VaultJWTRole{
		{
			BoundAudiences: []string{"k8s.relay.sh/metadata-api/v1"},
			ClaimMappings: map[string]string{
				"relay.sh/domain-id": "domain_id",
				"relay.sh/tenant-id": "tenant_id",
			},
			Name:          "tenant",
			RoleType:      "jwt",
			TokenPolicies: []string{"metadata-api-tenant"},
			TokenType:     "batch",
			UserClaim:     "sub",
		},
	}

	err = vi.vaultSystemManager.CreateTransitKey(vi.vaultCoreConfig.TransitPath, vi.vaultCoreConfig.TransitKey)
	if err != nil {
		return err
	}

	if err := vi.vaultInitializationManager.InitializeVault(ctx,
		&model.VaultInitializationData{
			JWTRoles:        jwtRoles,
			KubernetesRoles: kubernetesRoles,
			Policies:        policies,
		}); err != nil {
		return err
	}

	return nil
}

func NewVaultInitializer(
	vaultConfig *vaultutil.VaultConfig, vaultCoreConfig *VaultCoreConfig,
	vaultInitializationManager model.VaultInitializationManager,
	vaultSystemManager model.VaultSystemManager) *VaultInitializer {

	return &VaultInitializer{
		vaultConfig:                vaultConfig,
		vaultCoreConfig:            vaultCoreConfig,
		vaultInitializationManager: vaultInitializationManager,
		vaultSystemManager:         vaultSystemManager,
	}
}
