package prometheus

import (
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/puppetlabs/horsehead/v2/instrumentation/errors"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

type Prometheus struct {
	namespace string
}

func (p *Prometheus) NewCounter(name string, opts collectors.CounterOptions) (collectors.Counter, error) {
	c := prom.NewCounterVec(prom.CounterOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      opts.Description,
	}, opts.Labels)

	if err := prom.Register(c); err != nil {
		return nil, errors.NewMetricsUnknownError("prometheus").WithCause(err)
	}

	return &Counter{vector: c}, nil
}

func (p *Prometheus) NewTimer(name string, opts collectors.TimerOptions) (collectors.Timer, error) {
	observer := prom.NewHistogramVec(prom.HistogramOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      opts.Description,
		Buckets:   opts.HistogramBoundaries,
	}, opts.Labels)

	if err := prom.Register(observer); err != nil {
		return nil, errors.NewMetricsUnknownError("prometheus").WithCause(err)
	}

	t := NewTimer(observer)

	return t, nil
}

func (p *Prometheus) NewDurationMiddleware(name string, opts collectors.DurationMiddlewareOptions) (collectors.DurationMiddleware, error) {
	observer := prom.NewHistogramVec(prom.HistogramOpts{
		Namespace: p.namespace,
		Name:      name,
		Help:      opts.Description,
		Buckets:   opts.HistogramBoundaries,
	}, opts.Labels)

	if err := prom.Register(observer); err != nil {
		return nil, errors.NewMetricsUnknownError("prometheus").WithCause(err)
	}

	d := &DurationMiddleware{vector: observer}

	return d, nil
}

func (p *Prometheus) NewHandler() http.Handler {
	return promhttp.Handler()
}

func New(namespace string) *Prometheus {
	return &Prometheus{
		namespace: namespace,
	}
}
