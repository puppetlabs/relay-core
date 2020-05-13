package tenant

import (
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
)

type controllerObservations struct {
	mets *metrics.Metrics
}

func newControllerObservations(mets *metrics.Metrics) *controllerObservations {
	return &controllerObservations{
		mets: mets,
	}
}
