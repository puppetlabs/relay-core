package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VaultAgentDeps struct {
	Core           *obj.Core
	ConfigMap      *corev1.ConfigMap
	ServiceAccount *corev1.ServiceAccount
	TokenSecret    *corev1.Secret
}

func (vd *VaultAgentDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	return true, nil
}

func NewVaultAgentDeps(c *obj.Core) *VaultAgentDeps {
	return &VaultAgentDeps{Core: c}
}
