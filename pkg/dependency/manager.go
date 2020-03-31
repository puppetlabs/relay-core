package dependency

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/storage"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	nebclientset "github.com/puppetlabs/nebula-tasks/pkg/generated/clientset/versioned"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	tekclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type DependencyManager struct {
	Config        *config.WorkflowControllerConfig
	KubeClient    kubernetes.Interface
	NebulaClient  nebclientset.Interface
	TektonClient  tekclientset.Interface
	SecretsClient SecretAuthAccessManager
	StorageClient storage.BlobStore
	Metrics       *metrics.Metrics
}

func NewDependencyManager(kcfg *rest.Config, cfg *config.WorkflowControllerConfig, vc *vault.VaultAuth, bs storage.BlobStore, mets *metrics.Metrics) (*DependencyManager, error) {
	kubeclient, err := kubernetes.NewForConfig(kcfg)
	if err != nil {
		return nil, err
	}

	nebclient, err := nebclientset.NewForConfig(kcfg)
	if err != nil {
		return nil, err
	}

	tekclient, err := tekclientset.NewForConfig(kcfg)
	if err != nil {
		return nil, err
	}

	d := &DependencyManager{
		KubeClient:    kubeclient,
		NebulaClient:  nebclient,
		TektonClient:  tekclient,
		SecretsClient: vc,
		StorageClient: bs,
		Config:        cfg,
		Metrics:       mets,
	}

	return d, nil
}

type SecretAuthAccessManager interface {
	GrantScopedAccess(ctx context.Context, workflowID, namespace, serviceAccount string) (*secrets.AccessGrant, error)
	RevokeScopedAccess(ctx context.Context, namespace string) error
}
