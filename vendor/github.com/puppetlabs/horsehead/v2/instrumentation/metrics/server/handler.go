package server

import (
	"net/http"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
)

// Handler returns metrics in the style of the configured delegate (i.e. prometheus)
type Handler struct {
	delegate http.Handler
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.delegate.ServeHTTP(w, r)
}

// NewHandler returns a new http.Handler
func NewHandler(m *metrics.Metrics) http.Handler {
	return Handler{m.Handler()}
}
