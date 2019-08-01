package gcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

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

	exists, err := k.clusterExists(ctx, stateStoreURL)
	if err != nil {
		return nil, err
	}

	if !exists {
		if err := k.create(ctx, stateStoreURL); err != nil {
			return nil, err
		}
	} else {
		// TODO: we still need to serialize a kops resource file in order to update the cluster
		// if err := k.update(ctx, stateStoreURL); err != nil {
		// 	return nil, err
		// }
	}

	if err := k.exportKubeconfig(ctx, stateStoreURL); err != nil {
		return nil, err
	}

	if err := k.waitForClusterReady(ctx, stateStoreURL); err != nil {
		return nil, err
	}

	// TODO: cleanup temporary credentials

	return &models.K8sClusterState{Status: models.ClusterStatusRunning}, nil
}

func (k *K8sClusterManager) clusterExists(ctx context.Context, stateStoreURL *url.URL) (bool, errors.Error) {
	stdout := os.Stdout
	stderr := os.Stderr

	buf := &bytes.Buffer{}
	mw := io.MultiWriter(buf, stderr)

	kcmd := exec.CommandContext(
		ctx,
		"kops",
		"--state", stateStoreURL.String(),
		"get", "cluster",
		k.spec.ClusterName,
	)

	kcmd.Env = append(os.Environ(),
		"KOPS_FEATURE_FLAGS=AlphaAllowGCE",
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", k.support.CredentialsFile))

	kcmd.Stdout = stdout
	kcmd.Stderr = mw

	cerr := kcmd.Run()
	if cerr != nil {
		if strings.Contains(buf.String(), "cluster not found") {
			return false, nil
		}

		return false, errors.NewK8sProvisionerKopsExecError().WithCause(cerr)
	}

	return true, nil
}

func (k K8sClusterManager) exportKubeconfig(ctx context.Context, stateStoreURL *url.URL) errors.Error {
	kcmd := exec.CommandContext(
		ctx,
		"kops",
		"--state", stateStoreURL.String(),
		"export", "kubecfg",
		k.spec.ClusterName,
	)

	kcmd.Env = append(os.Environ(),
		"KOPS_FEATURE_FLAGS=AlphaAllowGCE",
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", k.support.CredentialsFile))

	kcmd.Stdout = os.Stdout
	kcmd.Stderr = os.Stderr

	if err := kcmd.Run(); err != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	return nil
}

func (k *K8sClusterManager) create(ctx context.Context, stateStoreURL *url.URL) errors.Error {
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

	kcmd.Stdout = os.Stdout
	kcmd.Stderr = os.Stderr

	if err := kcmd.Run(); err != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	return nil
}

func (k *K8sClusterManager) update(ctx context.Context, stateStoreURL *url.URL) errors.Error {
	nodeCount := k.spec.NodeCount
	if nodeCount == 0 {
		nodeCount = DefaultNodeCount
	}

	kcmd := exec.CommandContext(
		ctx,
		"kops",
		"update", "cluster", "--yes",
		"--state", stateStoreURL.String(),
		"--name", k.spec.ClusterName,
		"--zones", k.spec.Zones[0],
		"--project", k.spec.Project,
		"--node-count", strconv.Itoa(nodeCount),
	)

	kcmd.Env = append(os.Environ(),
		"KOPS_FEATURE_FLAGS=AlphaAllowGCE",
		fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", k.support.CredentialsFile))

	kcmd.Stdout = os.Stdout
	kcmd.Stderr = os.Stderr

	if err := kcmd.Run(); err != nil {
		return errors.NewK8sProvisionerKopsExecError().WithCause(err)
	}

	return nil
}
func (k *K8sClusterManager) waitForClusterReady(ctx context.Context, stateStoreURL *url.URL) errors.Error {
	stdout := os.Stdout
	stderr := os.Stderr

	buf := &bytes.Buffer{}
	mw := io.MultiWriter(buf, stdout)

	for {
		kcmd := exec.CommandContext(
			ctx,
			"kops",
			"--state", stateStoreURL.String(),
			"validate", "cluster",
			k.spec.ClusterName,
		)

		kcmd.Env = append(os.Environ(),
			"KOPS_FEATURE_FLAGS=AlphaAllowGCE",
			fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", k.support.CredentialsFile))

		kcmd.Stdout = mw
		kcmd.Stderr = stderr

		select {
		case <-ctx.Done():
			return errors.NewK8sProvisionerReadinessTimeoutError()
		case <-time.After(time.Second * 10):
		}

		if err := kcmd.Run(); err != nil {
			continue
		}

		if !strings.Contains(buf.String(), "is ready") {
			continue
		}

		break
	}

	return nil
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
