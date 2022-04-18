package vault

import (
	"fmt"
	"path"

	"github.com/hashicorp/hcl2/gohcl"
	"github.com/hashicorp/hcl2/hclwrite"
)

// vaultPolicy is a subset of the vault.Policy model. We created this here for
// 2 reasons: vault is a big module that we want to prevent pulling in as much
// as possible (the sdk and api are usually okay) and the model isn't made for
// encoding, so it doesn't include the proper hcl tags.
type vaultPolicy struct {
	Paths []vaultPolicyPath `hcl:"path,block"`
}

type vaultPolicyPath struct {
	Name         string   `hcl:"name,label"`
	Capabilities []string `hcl:"capabilities"`
}

type vaultPolicyGenerator struct {
	AuthJWTAccessor string
	LogServicePath  string
	TenantPath      string
	TransitPath     string
	TransitKey      string
}

func (g *vaultPolicyGenerator) operatorPolicy() []byte {
	policy := vaultPolicy{
		Paths: []vaultPolicyPath{
			{
				Name:         path.Join(g.TransitPath, "encrypt", g.TransitKey),
				Capabilities: []string{"update"},
			},
		},
	}

	return g.generate(&policy)
}

func (g *vaultPolicyGenerator) logServicePolicy() []byte {
	policy := vaultPolicy{
		Paths: []vaultPolicyPath{
			{
				Name:         path.Join("sys", "mounts"),
				Capabilities: []string{"read"},
			},
			{
				Name:         path.Join(g.LogServicePath, "data", "logs", "*"),
				Capabilities: []string{"create", "read", "update"},
			},
			{
				Name:         path.Join(g.LogServicePath, "data", "contexts", "*"),
				Capabilities: []string{"create", "read", "update"},
			},
			{
				Name:         path.Join(g.LogServicePath, "metadata", "logs", "*"),
				Capabilities: []string{"list", "delete"},
			},
			{
				Name:         path.Join(g.LogServicePath, "metadata", "contexts", "*"),
				Capabilities: []string{"list", "delete"},
			},
		},
	}

	return g.generate(&policy)
}

func (g *vaultPolicyGenerator) metadataAPIPolicy() []byte {
	policy := vaultPolicy{
		Paths: []vaultPolicyPath{
			{
				Name:         path.Join(g.TransitPath, "decrypt", g.TransitKey),
				Capabilities: []string{"update"},
			},
		},
	}

	return g.generate(&policy)
}

func (g *vaultPolicyGenerator) metadataAPITenantPolicy() []byte {
	tenantEntity := fmt.Sprintf("{{identity.entity.aliases.%s.metadata.tenant_id}}", g.AuthJWTAccessor)
	domainEntity := fmt.Sprintf("{{identity.entity.aliases.%s.metadata.domain_id}}", g.AuthJWTAccessor)

	policy := vaultPolicy{
		Paths: []vaultPolicyPath{
			{
				Name:         path.Join(g.TenantPath, "metadata", "workflows", tenantEntity, "*"),
				Capabilities: []string{"list"},
			},
			{
				Name:         path.Join(g.TenantPath, "data", "workflows", tenantEntity, "*"),
				Capabilities: []string{"read"},
			},
			{
				Name:         path.Join(g.TenantPath, "metadata", "connections", domainEntity, "*"),
				Capabilities: []string{"list"},
			},
			{
				Name:         path.Join(g.TenantPath, "data", "connections", domainEntity, "*"),
				Capabilities: []string{"read"},
			},
		},
	}

	return g.generate(&policy)
}

func (g *vaultPolicyGenerator) generate(policy *vaultPolicy) []byte {
	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(policy, f.Body())

	return f.Bytes()
}
