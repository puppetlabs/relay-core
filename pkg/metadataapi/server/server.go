package server

import (
	"context"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/horsehead/v2/netutil"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

// Server listens on a host and port and contains sub routers to route
// requests to.
type Server struct {
	// bindAddr is the address and port to listen on
	bindAddr string
	logger   logging.Logger
	managers op.ManagerFactory

	// secretsHander handles requests to secrets on the /secrets/* path
	secretsHandler http.Handler

	// specHandler handles requests to the given task spec on /spec/* path
	specHandler http.Handler

	// specsHandler handles requests to specs on the /specs/* path
	specsHandler http.Handler

	// outputsHandler handles requests for setting and getting task outputs
	// on the /outputs/* path
	outputsHandler http.Handler

	// stateHandler handles requests for setting and getting task state
	// on the /state/* path
	stateHandler http.Handler

	// conditionalsHandler handles requests for evaluating whether or not a step
	// condition is true
	conditionalsHandler http.Handler

	// healthCheckHandler handles requests to check the readiness and health of
	// the metadata server
	healthCheckHandler http.Handler
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var head string

	head, r.URL.Path = shiftPath(r.URL.Path)

	switch head {
	case "secrets":
		s.secretsHandler.ServeHTTP(w, r)
	case "spec":
		// New specs handler, uses the spec of the request IP for lookup.
		s.specHandler.ServeHTTP(w, r)
	case "specs":
		// Old specs handler, requires a unique task ID for lookup.
		//
		// TODO: Deprecate.
		s.specsHandler.ServeHTTP(w, r)
	case "outputs":
		s.outputsHandler.ServeHTTP(w, r)
	case "state":
		s.stateHandler.ServeHTTP(w, r)
	case "conditions":
		s.conditionalsHandler.ServeHTTP(w, r)
	case "healthz":
		s.healthCheckHandler.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Run created a new network listener and handles shutdowns when the context closes.
func (s *Server) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.bindAddr)
	if err != nil {
		return errors.NewServerRunError().WithCause(err)
	}

	ln = netutil.NewTCPKeepAliveListener(ln.(*net.TCPListener))
	ln = netutil.NewContextListener(ctx, ln)

	srv := &http.Server{Handler: s}

	defer srv.Shutdown(ctx)

	if err := srv.Serve(ln); err != nil {
		return errors.NewServerRunError().WithCause(err)
	}

	return nil
}

// New returns a new Server that routes requests to sub-routers. It is also responsible
// for binding to a listener and is the first point of entry for http requests.
func New(cfg *config.MetadataServerConfig, managers op.ManagerFactory) *Server {
	withManagers := middleware.ManagerFactoryMiddleware(managers)
	withTask := middleware.TaskMetadataMiddleware

	secrets := withManagers(&secretsHandler{
		logger: cfg.Logger,
	})

	spec := withManagers(withTask(&specHandler{
		logger: cfg.Logger,
	}))

	specs := withManagers(withTask(&specsHandler{
		logger: cfg.Logger,
	}))

	conditionals := withManagers(withTask(&conditionalsHandler{
		logger: cfg.Logger,
	}))

	outputs := withManagers(withTask(&outputsHandler{
		logger: cfg.Logger,
	}))

	state := withManagers(withTask(&stateHandler{
		logger: cfg.Logger,
	}))

	return &Server{
		bindAddr:            cfg.BindAddr,
		logger:              cfg.Logger,
		managers:            managers,
		secretsHandler:      secrets,
		specHandler:         spec,
		specsHandler:        specs,
		outputsHandler:      outputs,
		stateHandler:        state,
		conditionalsHandler: conditionals,
		healthCheckHandler:  &healthCheckHandler{},
	}
}

// shiftPath takes a URI path and returns the first segment as the head
// and the remaining segments as the tail.
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], ""
	}

	return p[1:i], p[i:]
}
