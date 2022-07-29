package dependency

import (
	"log"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/leg/storage"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	jose "gopkg.in/square/go-jose.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(
		scheme.AddToScheme,
		relayv1beta1.AddToScheme,
		tektonv1alpha1.AddToScheme,
		tektonv1beta1.AddToScheme,
		servingv1.AddToScheme,
	)
	AddToScheme = SchemeBuilder.AddToScheme

	Scheme = runtime.NewScheme()
)

func init() {
	if err := SchemeBuilder.AddToScheme(Scheme); err != nil {
		panic(err)
	}
}

type DependencyManager struct {
	Manager       manager.Manager
	Config        *config.WorkflowControllerConfig
	KubeClient    kubernetes.Interface
	VaultClient   *vaultapi.Client
	JWTSigner     jose.Signer
	StorageClient storage.BlobStore
}

func NewDependencyManager(cfg *config.WorkflowControllerConfig, kcc *rest.Config, vc *vaultapi.Client, jwtSigner jose.Signer, bs storage.BlobStore) (*DependencyManager, error) {
	mgr, err := ctrl.NewManager(kcc, ctrl.Options{
		Scheme:             Scheme,
		MetricsBindAddress: "0",
		Port:               cfg.WebhookServerPort,
		CertDir:            cfg.WebhookServerKeyDir,
	})
	if err != nil {
		log.Fatal("Unable to create new manager", err)
	}

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
	}
	return d, nil
}
