package dependency

import (
	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/install/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(
		scheme.AddToScheme,
		relayv1beta1.AddToScheme,
		installerv1alpha1.AddToScheme,
	)
	AddToScheme = SchemeBuilder.AddToScheme

	Scheme = runtime.NewScheme()
)

func init() {
	if err := SchemeBuilder.AddToScheme(Scheme); err != nil {
		panic(err)
	}
}

type Manager struct {
	Manager    manager.Manager
	Config     *config.InstallerControllerConfig
	KubeClient kubernetes.Interface
}

func NewManager(cfg *config.InstallerControllerConfig, kcc *rest.Config) (*Manager, error) {
	log := ctrl.Log.WithName("relay-installer")

	mgr, err := ctrl.NewManager(kcc, ctrl.Options{
		Scheme: Scheme,
	})
	if err != nil {
		log.Error(err, "Unable to create new manager")

		return nil, err
	}

	kc, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, err
	}

	d := &Manager{
		Manager:    mgr,
		Config:     cfg,
		KubeClient: kc,
	}
	return d, nil
}
