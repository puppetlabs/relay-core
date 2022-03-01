package vault

import (
	"net/url"

	vaultutil "github.com/puppetlabs/leg/vaultutil/pkg/vault"
	"github.com/spf13/viper"
)

const (
	DefaultJWTAuthPath        = "auth/jwt"
	DefaultJWTMount           = "jwt"
	DefaultKubernetesAuthPath = "auth/kubernetes"
	DefaultKubernetesMount    = "kubernetes"
	DefaultVaultURL           = "http://localhost:8200"

	EnvironmentPrefix = "relay_operator_vault_init"

	VaultAddrConfigOption               = "vault_addr"
	VaultJWTAuthPathConfigOption        = "vault_jwt_auth_path"
	VaultJWTMountConfigOption           = "vault_jwt_mount"
	VaultJWTPublicKeyConfigOption       = "vault_jwt_public_key"
	VaultKubernetesAuthPathConfigOption = "vault_kubernetes_auth_path"
	VaultKubernetesMountConfigOption    = "vault_kubernetes_mount"
	VaultNameConfigOption               = "vault_name"
	VaultNamespaceConfigOption          = "vault_namespace"
	VaultServiceAccountConfigOption     = "vault_service_account"
	VaultTokenConfigOption              = "vault_token"
	VaultTransitMountConfigOption       = "vault_transit_mount"
	VaultUnsealKeyConfigOption          = "vault_unseal_key"

	LogServicePathConfigOption            = "log_service_path"
	LogServiceVaultAgentRoleConfigOption  = "log_service_vault_agent_role"
	MetadataAPIVaultAgentRoleConfigOption = "metadata_api_vault_agent_role"
	OperatorVaultAgentRoleConfigOption    = "operator_vault_agent_role"
	TenantPathConfigOption                = "tenant_path"
	TransitKeyConfigOption                = "transit_key"
	TransitPathConfigOption               = "transit_path"
)

type VaultCoreConfig struct {
	LogServicePath            string
	LogServiceVaultAgentRole  string
	MetadataAPIVaultAgentRole string
	OperatorVaultAgentRole    string
	TenantPath                string
	TransitKey                string
	TransitPath               string
}

func NewConfig() (*vaultutil.VaultConfig, *VaultCoreConfig, error) {
	viper.SetEnvPrefix(EnvironmentPrefix)
	viper.AutomaticEnv()

	viper.SetDefault(VaultAddrConfigOption, DefaultVaultURL)
	viper.SetDefault(VaultJWTAuthPathConfigOption, DefaultJWTAuthPath)
	viper.SetDefault(VaultJWTMountConfigOption, DefaultJWTMount)
	viper.SetDefault(VaultKubernetesAuthPathConfigOption, DefaultKubernetesAuthPath)
	viper.SetDefault(VaultKubernetesMountConfigOption, DefaultKubernetesMount)

	vaultURL, err := url.Parse(viper.GetString(VaultAddrConfigOption))
	if err != nil {
		return nil, nil, err
	}

	vaultConfig := &vaultutil.VaultConfig{
		JWTAuthPath:        viper.GetString(VaultJWTAuthPathConfigOption),
		JWTMount:           viper.GetString(VaultJWTMountConfigOption),
		JWTPublicKey:       viper.GetString(VaultJWTPublicKeyConfigOption),
		KubernetesAuthPath: viper.GetString(VaultKubernetesAuthPathConfigOption),
		KubernetesMount:    viper.GetString(VaultKubernetesMountConfigOption),
		Name:               viper.GetString(VaultNameConfigOption),
		Namespace:          viper.GetString(VaultNamespaceConfigOption),
		ServiceAccount:     viper.GetString(VaultServiceAccountConfigOption),
		Token:              viper.GetString(VaultTokenConfigOption),
		UnsealKey:          viper.GetString(VaultUnsealKeyConfigOption),
		VaultAddr:          vaultURL,
	}

	vaultCoreConfig := &VaultCoreConfig{
		LogServicePath:            viper.GetString(LogServicePathConfigOption),
		LogServiceVaultAgentRole:  viper.GetString(LogServiceVaultAgentRoleConfigOption),
		MetadataAPIVaultAgentRole: viper.GetString(MetadataAPIVaultAgentRoleConfigOption),
		OperatorVaultAgentRole:    viper.GetString(OperatorVaultAgentRoleConfigOption),
		TenantPath:                viper.GetString(TenantPathConfigOption),
		TransitKey:                viper.GetString(TransitKeyConfigOption),
		TransitPath:               viper.GetString(TransitPathConfigOption),
	}

	return vaultConfig, vaultCoreConfig, nil
}
