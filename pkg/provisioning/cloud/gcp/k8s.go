package gcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/cloud/gcp/support"
	"github.com/puppetlabs/nebula-tasks/pkg/provisioning/models"
)

const DefaultNodeCount = 2

// K8sClusterManager creates, updates or deletes clusters running inside GCP
type K8sClusterManager struct {
	spec    *models.K8sProvisionerSpec
	support *support.KopsSupport
}

// Synchronize ensures a state bucket exists, then translates the spec into a kops configuration
// to apply and boot the k8s cluster. This method makes sure the cluster is in the desired state
// when the workflow runs.
func (k *K8sClusterManager) Synchronize(ctx context.Context) (*models.K8sClusterState, errors.Error) {
	stateStoreURL, err := k.support.StateStoreURL(ctx)
	if err != nil {
		return nil, errors.NewK8sProvisionerStateStoreError().WithCause(err)
	}

	stdout := os.Stdout
	stderr := os.Stderr

	gcmd := exec.CommandContext(ctx, "gcloud", "auth", "activate-service-account",
		"--key-file", k.support.CredentialsFile)

	gcmd.Stdout = stdout
	gcmd.Stderr = stderr

	if err := gcmd.Run(); err != nil {
		return nil, errors.NewK8sProvisionerAuthError().WithCause(err)
	}

	nodeCount := k.spec.NodeCount
	if nodeCount == 0 {
		nodeCount = DefaultNodeCount
	}

	kcmd := exec.CommandContext(
		ctx,
		"kops",
		"create", "cluster", "--yes",
		"--state", stateStoreURL.String(),
		"--name", k.spec.ClusterName,
		"--zones", k.spec.Zones[0],
		"--project", k.spec.Project,
		"--node-count", strconv.Itoa(nodeCount),
	)

	kcmd.Env = append(os.Environ(),
		"KOPS_FEATURE_FLAGS=AlphaAllowGCE",
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", k.support.CredentialsFile))

	kcmd.Stdout = stdout
	kcmd.Stderr = stderr

	if err := kcmd.Run(); err != nil {
		return nil, errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	// TODO: validate cluster
	// TODO: wait for cluster to come online
	// TODO: cleanup temporary credentials

	return &models.K8sClusterState{Status: models.ClusterStatusRunning}, nil
}

func (k *K8sClusterManager) clusterExists(name string) (bool, error) {
	return true, nil
}

// NewK8sClusterManager returns a new K8sClusterManager or an error
func NewK8sClusterManager(spec *models.K8sProvisionerSpec) (*K8sClusterManager, errors.Error) {
	ks, err := support.NewKopsSupport(spec.Project, spec.Credentials, spec.StateStoreName)
	if err != nil {
		return nil, errors.NewK8sProvisionerClusterManagerSetupError().WithCause(err)
	}

	return &K8sClusterManager{
		spec:    spec,
		support: ks,
	}, nil
}
