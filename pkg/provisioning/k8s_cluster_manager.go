package provisioning

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs/client"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
)

const KubeconfigOutputKey = "kubeconfig-file"

type K8sClusterAdapter interface {
	ProvisionCluster(ctx context.Context) errors.Error
	GetCluster(ctx context.Context) (*models.K8sCluster, errors.Error)
	GetKubeconfig(ctx context.Context) (io.Reader, errors.Error)
}

type K8sClusterManager interface {
	Synchronize(context.Context) (*models.K8sCluster, errors.Error)
	SaveKubeconfig(context.Context) errors.Error
}

type K8sClusterManagerConfig struct {
	Spec          *models.K8sProvisionerSpec
	Workdir       string
	OutputsClient client.OutputsClient
}

func NewK8sClusterManagerFromSpec(cfg K8sClusterManagerConfig) (K8sClusterManager, errors.Error) {
	platform, ok := PlatformMapping[strings.ToLower(cfg.Spec.Provider)]
	if !ok {
		return nil, errors.NewK8sProvisionerUnknownProvider(cfg.Spec.Provider)
	}

	adapter := PlatformAdapters[platform](cfg.Spec, cfg.Workdir)

	return NewK8sClusterManager(cfg, adapter), nil
}

// k8sClusterManager creates, updates or deletes clusters
type k8sClusterManager struct {
	spec          *models.K8sProvisionerSpec
	workdir       string
	adapter       K8sClusterAdapter
	outputsClient client.OutputsClient
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

// SaveKubeconfig will save the kubeconfig file for kubectl into the Nebula outputs storage
// for subsequent tasks to use.
func (k *k8sClusterManager) SaveKubeconfig(ctx context.Context) errors.Error {
	r, err := k.adapter.GetKubeconfig(ctx)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(r); err != nil {
		return errors.NewK8sProvisionerKubeconfigReadError().WithCause(err)
	}

	if err := k.outputsClient.SetOutput(ctx, KubeconfigOutputKey, buf.String()); err != nil {
		return errors.NewOutputsClientSetFailed().WithCause(err)
	}

	return nil
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
func NewK8sClusterManager(cfg K8sClusterManagerConfig, adapter K8sClusterAdapter) *k8sClusterManager {
	return &k8sClusterManager{
		spec:          cfg.Spec,
		workdir:       cfg.Workdir,
		adapter:       adapter,
		outputsClient: cfg.OutputsClient,
	}
}
