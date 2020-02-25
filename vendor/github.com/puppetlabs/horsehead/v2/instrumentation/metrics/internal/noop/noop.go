package noop

import (
	"net/http"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

type Noop struct{}

func (n *Noop) NewCounter(name string, opts collectors.CounterOptions) (collectors.Counter, error) {
	return &Counter{}, nil
}

func (n *Noop) NewTimer(name string, opts collectors.TimerOptions) (collectors.Timer, error) {
	return &Timer{}, nil
}

func (n *Noop) NewDurationMiddleware(name string, opts collectors.DurationMiddlewareOptions) (collectors.DurationMiddleware, error) {
	return &DurationMiddleware{}, nil
}

func (n *Noop) NewHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

func New() *Noop {
	return &Noop{}
}
