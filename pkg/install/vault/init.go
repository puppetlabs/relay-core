package vault

import (
	"context"
	"fmt"

	"github.com/puppetlabs/leg/vaultutil/pkg/model"
	vaultutil "github.com/puppetlabs/leg/vaultutil/pkg/vault"
)

type VaultInitializer struct {
	vaultConfig        *vaultutil.VaultConfig
	vaultCoreConfig    *VaultCoreConfig
	vaultSystemManager model.VaultSystemManager
}

func (vi *VaultInitializer) InitializeVault(ctx context.Context) error {
	credentials := &model.VaultKeys{
		RootToken:  vi.vaultConfig.Token,
		UnsealKeys: []string{vi.vaultConfig.UnsealKey},
	}

	var err error
	if credentials.RootToken == "" || credentials.UnsealKeys[0] == "" {
		credentials, err = vi.vaultSystemManager.GetCredentials(ctx)
		if err != nil {
			return err
		}

		if credentials == nil {
			credentials, err = vi.vaultSystemManager.Initialize(ctx)
			if err != nil {
				return err
			}
		}
	}

	err = vi.vaultSystemManager.Unseal(credentials)
	if err != nil {
		return err
	}

	vi.vaultSystemManager.SetToken(credentials)

	err = vi.vaultSystemManager.EnableKubernetesAuth()
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.ConfigureKubernetesAuth(ctx)
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.EnableJWTAuth()
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.ConfigureJWTAuth(ctx)
	if err != nil {
		return err
	}

	secretEngines := []*model.VaultSecretEngine{
		{
			Name: vi.vaultCoreConfig.TenantPath,
			Type: model.VaultSecretEngineTypeKVV2,
		},
		{
			Name: vi.vaultCoreConfig.TransitPath,
			Type: model.VaultSecretEngineTypeTransit,
		},
	}

	if vi.vaultCoreConfig.LogServicePath != "" {
		secretEngines = append(secretEngines, &model.VaultSecretEngine{
			Name: vi.vaultCoreConfig.LogServicePath,
			Type: model.VaultSecretEngineTypeKVV2,
		})
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

	err = vi.vaultSystemManager.EnableSecretEngines(secretEngines)
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.CreateTransitKey(vi.vaultCoreConfig.TransitPath, vi.vaultCoreConfig.TransitKey)
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.PutPolicies(policies)
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.ConfigureJWTAuthRoles(jwtRoles)
	if err != nil {
		return err
	}

	err = vi.vaultSystemManager.ConfigureKubernetesAuthRoles(kubernetesRoles)
	if err != nil {
		return err
	}

	return nil
}

func NewVaultInitializer(
	vaultSystemManager model.VaultSystemManager,
	vaultConfig *vaultutil.VaultConfig, vaultCoreConfig *VaultCoreConfig) *VaultInitializer {

	return &VaultInitializer{
		vaultConfig:        vaultConfig,
		vaultCoreConfig:    vaultCoreConfig,
		vaultSystemManager: vaultSystemManager,
	}
}
