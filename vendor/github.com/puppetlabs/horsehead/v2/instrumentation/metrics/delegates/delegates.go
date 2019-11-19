package delegates

import (
	"errors"
	"net/http"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/internal/prometheus"
)

// Delegate is an interface metrics collectors implement (i.e. prometheus)
type Delegate interface {
	NewCounter(name string, opts collectors.CounterOptions) (collectors.Counter, error)
	NewTimer(name string, opts collectors.TimerOptions) (collectors.Timer, error)
	NewDurationMiddleware(name string, opts collectors.DurationMiddlewareOptions) (collectors.DurationMiddleware, error)
	NewHandler() http.Handler
}

// DelegateType is a string representation of all the available metric backend delegates
type DelegateType string

const (
	// PrometheusDelegate is a const that represents the prometheus backend
	PrometheusDelegate DelegateType = "prometheus"
)

// New looks up t and returns a new Delegate matching that type
func New(namespace string, t DelegateType) (Delegate, error) {
	switch t {
	case PrometheusDelegate:
		return prometheus.New(namespace), nil
	}

	return nil, errors.New("no delegate found")
}
