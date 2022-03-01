package config

import (
	"github.com/puppetlabs/leg/instrumentation/alerts"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
)

type InstallerControllerConfig struct {
	Environment             string
	MaxConcurrentReconciles int
	Namespace               string
	AlertsDelegate          alerts.DelegateFunc
}

func (c *InstallerControllerConfig) Capturer() trackers.Capturer {
	alertsDelegate := c.AlertsDelegate
	if alertsDelegate == nil {
		alertsDelegate = alerts.NoDelegate
	}

	a := alerts.NewAlerts(alertsDelegate, alerts.Options{
		Environment: c.Environment,
	})

	return a.NewCapturer().WithNewTrace().WithAppPackages([]string{"github.com/puppetlabs/relay-core"})
}

func NewDefaultInstallerControllerConfig() *InstallerControllerConfig {
	return &InstallerControllerConfig{
		MaxConcurrentReconciles: 1,
		Namespace:               "default",
	}
}
