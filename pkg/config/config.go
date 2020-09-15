package config

import (
	"net/url"

	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkflowControllerConfig is the configuration object used to
// configure the Workflow controller.
type WorkflowControllerConfig struct {
	Environment             string
	Standalone              bool
	Namespace               string
	ImagePullSecret         string
	MaxConcurrentReconciles int
	MetadataAPIURL          *url.URL
	VaultTransitPath        string
	VaultTransitKey         string
	WebhookServerPort       int
	WebhookServerKeyDir     string
	DynamicRBACBinding      bool
	ToolInjectionImage      string
	AlertsDelegate          alerts.DelegateFunc
}

func (c *WorkflowControllerConfig) Capturer() trackers.Capturer {
	alertsDelegate := c.AlertsDelegate
	if alertsDelegate == nil {
		alertsDelegate = alerts.NoDelegate
	}

	a := alerts.NewAlerts(alertsDelegate, alerts.Options{
		Environment: c.Environment,
	})

	return a.NewCapturer().
		WithNewTrace().
		WithAppPackages([]string{"github.com/puppetlabs/relay-core"})
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
