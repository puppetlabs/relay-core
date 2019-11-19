package prometheus

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

func convertLabels(labels []collectors.Label) prom.Labels {
	promLabels := prom.Labels{}
	for _, l := range labels {
		promLabels[l.Name] = l.Value
	}

	return promLabels
}
