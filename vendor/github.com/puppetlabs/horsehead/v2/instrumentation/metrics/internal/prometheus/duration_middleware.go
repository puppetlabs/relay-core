package prometheus

import (
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/puppetlabs/horsehead/v2/instrumentation/errors"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

type DurationMiddleware struct {
	vector prom.ObserverVec
}

func (d *DurationMiddleware) WithLabels(labels []collectors.Label) (collectors.DurationMiddleware, error) {
	vector, err := d.vector.CurryWith(convertLabels(labels))
	if err != nil {
		return nil, errors.NewMetricsUnknownError("prometheus").WithCause(err)
	}

	return &DurationMiddleware{
		vector: vector,
	}, nil
}

func (d *DurationMiddleware) Wrap(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(d.vector, next)
}
