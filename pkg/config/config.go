package config

import (
	"net/url"

	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkflowControllerConfig is the configuration object used to
// configure the Workflow controller.
type WorkflowControllerConfig struct {
	Namespace               string
	ImagePullSecret         string
	MaxConcurrentReconciles int
	MetadataAPIURL          *url.URL
	VaultTransitPath        string
	VaultTransitKey         string
}

func (c *WorkflowControllerConfig) ImagePullSecretKey() client.ObjectKey {
	namespace, name, err := cache.SplitMetaNamespaceKey(c.ImagePullSecret)
	if err != nil {
		name = c.ImagePullSecret
	}

	if namespace == "" {
		namespace = c.Namespace
	}

	return client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}
}

// K8sClusterProvisionerConfig is the configuration object to used
// to configure the Kubernetes provisioner task.
type K8sClusterProvisionerConfig struct {
	WorkDir string
}
