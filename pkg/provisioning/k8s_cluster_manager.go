package provisioning

import (
	"context"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
)

type K8sClusterAdapter interface {
	ProvisionCluster(ctx context.Context) errors.Error
	GetCluster(ctx context.Context) (*models.K8sCluster, errors.Error)
}

type K8sClusterManager interface {
	Synchronize(context.Context) (*models.K8sCluster, errors.Error)
}

func NewK8sClusterManagerFromSpec(spec *models.K8sProvisionerSpec) (K8sClusterManager, errors.Error) {
	platform, ok := PlatformMapping[spec.Provider]
	if !ok {
		return nil, errors.NewK8sProvisionerUnknownProvider(spec.Provider)
	}

	adapter, err := PlatformAdapters[platform](spec)
	if err != nil {
		return nil, errors.NewK8sProvisionerPlatformSetupError(spec.Provider).WithCause(err)
	}

	return NewK8sClusterManager(spec, adapter), nil
}

// k8sClusterManager creates, updates or deletes clusters
type k8sClusterManager struct {
	spec    *models.K8sProvisionerSpec
	adapter K8sClusterAdapter
}

// Synchronize ensures a state bucket exists, then translates the spec into a kops configuration
// to apply and boot the k8s cluster. This method makes sure the cluster is in the desired state
// when the workflow runs.
func (k *k8sClusterManager) Synchronize(ctx context.Context) (*models.K8sCluster, errors.Error) {
	if err := k.adapter.ProvisionCluster(ctx); err != nil {
		return nil, errors.NewK8sProvisionerClusterSynchronizationError().WithCause(err)
	}

	return k.untilStatus(ctx, models.ClusterStatusReady, func() (*models.K8sCluster, errors.Error) {
		cluster, err := k.adapter.GetCluster(ctx)
		if err != nil {
			return nil, err

		}

		return cluster, nil
	})

}

type untilCallbackFunc func() (*models.K8sCluster, errors.Error)

func (k *k8sClusterManager) untilStatus(ctx context.Context, exp models.ClusterStatus, fn untilCallbackFunc) (*models.K8sCluster, errors.Error) {
	var (
		cluster *models.K8sCluster
		err     errors.Error
	)

loop:
	for {
		select {
		case <-time.After(time.Second):
			cluster, err = fn()
			if err != nil {
				return nil, err
			}

			if cluster.Status != exp {
				continue
			}

			break loop
		case <-ctx.Done():
			return nil, errors.NewK8sProvisionerTimeoutError("timeout reached while waiting for cluster state change")
		}
	}

	return cluster, nil
}

// NewK8sClusterManager returns a new k8sClusterManager instance. Takes a spec and a platform adapter
// and synchronize makes relevant synchronization calls to the adapter.
func NewK8sClusterManager(spec *models.K8sProvisionerSpec, adapter K8sClusterAdapter) *k8sClusterManager {
	return &k8sClusterManager{
		spec:    spec,
		adapter: adapter,
	}
}
