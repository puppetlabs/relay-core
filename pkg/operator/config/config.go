package config

import (
	"net/url"

	"github.com/puppetlabs/leg/instrumentation/alerts"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
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
	RuntimeToolsImage       string
	VaultTransitPath        string
	VaultTransitKey         string
	WebhookServerPort       int
	WebhookServerKeyDir     string
	DynamicRBACBinding      bool
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
