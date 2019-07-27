package provisioning

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/cloud/gcp"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
)

type K8sClusterManager interface {
	Synchronize(context.Context) (*models.K8sClusterState, errors.Error)
}

func NewK8sClusterManagerFromSpec(spec *models.K8sProvisionerSpec) (K8sClusterManager, errors.Error) {
	switch spec.Provider {
	case "gcp":
		return gcp.NewK8sClusterManager(spec)
	}

	return nil, errors.NewK8sProvisionerUnknownProvider(spec.Provider)
}
