package op

import (
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type KubernetesManager interface {
	Client() kubernetes.Interface
	Namespace() string
}

type DefaultKubernetesManager struct {
	kubeclient kubernetes.Interface
	namespace  string
}

func (km DefaultKubernetesManager) Client() kubernetes.Interface {
	return km.kubeclient
}

func (km DefaultKubernetesManager) Namespace() string {
	if km.namespace == "" {
		return "default"
	}

	return km.namespace
}

func NewDefaultKubernetesManager(namespace string, kubeclient kubernetes.Interface) *DefaultKubernetesManager {
	return &DefaultKubernetesManager{
		kubeclient: kubeclient,
		namespace:  namespace,
	}
}

func NewKubeclientFromConfig(cfg *config.MetadataServerConfig) (kubernetes.Interface, errors.Error) {
	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.KubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: cfg.K8sMasterURL}},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		return nil, errors.NewKubernetesManagerSetupError().WithCause(err)
	}

	kubeclient, err := kubernetes.NewForConfig(kcc)
	if err != nil {
		return nil, errors.NewKubernetesManagerSetupError().WithCause(err)
	}

	return kubeclient, nil
}
