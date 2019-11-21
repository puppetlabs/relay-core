package collectors

import "net/http"

type DurationMiddlewareOptions struct {
	Description         string
	Labels              []string
	HistogramBoundaries []float64
}

type DurationMiddleware interface {
	WithLabels([]Label) (DurationMiddleware, error)
	Wrap(http.Handler) http.Handler
}
