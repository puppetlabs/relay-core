package app

import (
	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

type VaultAgentConfig struct {
	AutoAuth  *VaultAutoAuth   `hcl:"auto_auth,block"`
	Cache     *VaultCache      `hcl:"cache,block"`
	Listeners []*VaultListener `hcl:"listener,block"`
	Vault     *VaultServer     `hcl:"vault,block"`
}

type VaultAutoAuth struct {
	Method *VaultAutoAuthMethod `hcl:"method,block"`
}

type VaultAutoAuthMethod struct {
	Type      string            `hcl:"type,label"`
	MountPath string            `hcl:"mount_path"`
	Config    map[string]string `hcl:"config"`
}

type VaultCache struct {
	UseAutoAuthToken bool `hcl:"use_auto_auth_token"`
}

type VaultListener struct {
	Type        string `hcl:"type,label"`
	Address     string `hcl:"address"`
	TLSDisabled bool   `hcl:"tls_disabled"`
}

type VaultServer struct {
	Address string `hcl:"address"`
}

func generateVaultConfig(config *VaultAgentConfig) []byte {
	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(config, f.Body())

	return f.Bytes()
}
