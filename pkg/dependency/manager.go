package dependency

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/storage"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type DependencyManager struct {
	Manager       manager.Manager
	Config        *config.WorkflowControllerConfig
	KubeClient    kubernetes.Interface
	SecretsClient SecretAuthAccessManager
	StorageClient storage.BlobStore
	Metrics       *metrics.Metrics
}

func NewDependencyManager(mgr manager.Manager, cfg *config.WorkflowControllerConfig, vc *vault.VaultAuth, bs storage.BlobStore, mets *metrics.Metrics) (*DependencyManager, error) {
	kc, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	d := &DependencyManager{
		Manager:       mgr,
		Config:        cfg,
		KubeClient:    kc,
		SecretsClient: vc,
		StorageClient: bs,
		Metrics:       mets,
	}
	return d, nil
}

type SecretAuthAccessManager interface {
	GrantScopedAccess(ctx context.Context, workflowID, namespace, serviceAccount string) (*secrets.AccessGrant, error)
	RevokeScopedAccess(ctx context.Context, namespace string) error
}
