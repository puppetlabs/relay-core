package dependency

import (
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/storage"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type DependencyManager struct {
	Manager       manager.Manager
	Config        *config.WorkflowControllerConfig
	KubeClient    kubernetes.Interface
	VaultClient   *vaultapi.Client
	JWTSigner     jose.Signer
	StorageClient storage.BlobStore
	Metrics       *metrics.Metrics
}

func NewDependencyManager(mgr manager.Manager, cfg *config.WorkflowControllerConfig, vc *vaultapi.Client, jwtSigner jose.Signer, bs storage.BlobStore, mets *metrics.Metrics) (*DependencyManager, error) {
	kc, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	d := &DependencyManager{
		Manager:       mgr,
		Config:        cfg,
		KubeClient:    kc,
		VaultClient:   vc,
		JWTSigner:     jwtSigner,
		StorageClient: bs,
		Metrics:       mets,
	}
	return d, nil
}
