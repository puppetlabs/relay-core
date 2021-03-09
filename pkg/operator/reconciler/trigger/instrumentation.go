package trigger

import (
	"github.com/puppetlabs/leg/instrumentation/metrics"
)

type controllerObservations struct {
	mets *metrics.Metrics
}

func newControllerObservations(mets *metrics.Metrics) *controllerObservations {
	return &controllerObservations{
		mets: mets,
	}
}
