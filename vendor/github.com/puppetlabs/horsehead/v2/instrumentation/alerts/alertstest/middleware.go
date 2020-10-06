package alertstest

import (
	"net/http"

	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/internal/httputil"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
)

type Middleware struct {
	c *Capturer
}

func (m *Middleware) WithTags(tags ...trackers.Tag) trackers.Middleware {
	return m
}

func (m *Middleware) WithUser(u trackers.User) trackers.Middleware {
	return m
}

func (m *Middleware) Wrap(target http.Handler) http.Handler {
	return httputil.Wrap(target, httputil.WrapStatic(m.c))
}
