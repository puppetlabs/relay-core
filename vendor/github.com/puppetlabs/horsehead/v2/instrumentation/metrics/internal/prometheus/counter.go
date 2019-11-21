package prometheus

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/puppetlabs/horsehead/v2/instrumentation/errors"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

type Counter struct {
	vector   *prom.CounterVec
	delegate prom.Counter
	labels   []collectors.Label
}

func (c *Counter) WithLabels(labels []collectors.Label) (collectors.Counter, error) {
	delegate, err := c.vector.GetMetricWith(convertLabels(labels))
	if err != nil {
		return nil, errors.NewMetricsUnknownError("prometheus").WithCause(err)
	}

	return &Counter{
		vector:   c.vector,
		delegate: delegate,
		labels:   labels,
	}, nil
}

func (c *Counter) Add(n float64) {
	c.delegate.Add(n)
}

func (c *Counter) Inc() {
	c.delegate.Inc()
}
