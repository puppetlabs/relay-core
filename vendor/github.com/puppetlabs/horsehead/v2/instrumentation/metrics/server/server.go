package server

import (
	"context"
	"net"
	"net/http"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/netutil"
)

// Options are the Server configuration options
type Options struct {
	// BindAddr is the address:port to listen on
	BindAddr string

	// Path is the URI path to handle requests for.
	// Default is /
	Path string
}

// Server delegates http requests for metrics on a configured path to the Metrics
// collector.
type Server struct {
	bindAddr string
	m        *metrics.Metrics
	path     string
}

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.path {
		http.NotFound(w, r)

		return
	}

	handler := NewHandler(s.m)
	handler.ServeHTTP(w, r)
}

func (s Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.bindAddr)
	if err != nil {
		return err
	}

	ln = netutil.NewTCPKeepAliveListener(ln.(*net.TCPListener))
	ln = netutil.NewContextListener(ctx, ln)

	hs := &http.Server{Handler: s}
	err = hs.Serve(ln)
	hs.Shutdown(ctx)
	return err
}

// New returns a new Server
func New(m *metrics.Metrics, opts Options) *Server {
	path := opts.Path

	if path == "" {
		path = "/"
	}

	return &Server{
		bindAddr: opts.BindAddr,
		m:        m,
		path:     path,
	}
}
