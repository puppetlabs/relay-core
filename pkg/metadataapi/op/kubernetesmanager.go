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
}

type DefaultKubernetesManager struct {
	kubeclient kubernetes.Interface
}

func (km DefaultKubernetesManager) Client() kubernetes.Interface {
	return km.kubeclient
}

func NewDefaultKubernetesManager(kubeclient kubernetes.Interface) *DefaultKubernetesManager {
	return &DefaultKubernetesManager{
		kubeclient: kubeclient,
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
