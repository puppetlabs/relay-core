package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/leg/instrumentation/alerts/trackers"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
)

type Server struct {
	auth             middleware.Authenticator
	errorSensitivity errawr.ErrorSensitivity
	capturer         trackers.Capturer
	trustedProxyHops int
	schemaRegistry   validation.SchemaRegistry
}

func (s *Server) Route(r *mux.Router) {
	r.Use(middleware.WithErrorSensitivity(s.errorSensitivity))

	r.HandleFunc("/healthz", s.GetHealthz).Methods("GET")

	// This has a different set of middleware so bind it under a subrouter.
	api.NewServer(s.auth, api.WithSchemaRegistry(s.schemaRegistry)).
		Route(r.NewRoute().Subrouter())
}

type Option func(s *Server)

func WithErrorSensitivity(sensitivity errawr.ErrorSensitivity) Option {
	return func(s *Server) {
		s.errorSensitivity = sensitivity
	}
}

func WithCapturer(capturer trackers.Capturer) Option {
	return func(s *Server) {
		s.capturer = capturer
	}
}

func WithTrustedProxyHops(n int) Option {
	return func(s *Server) {
		s.trustedProxyHops = n
	}
}

func WithSchemaRegistry(r validation.SchemaRegistry) Option {
	return func(s *Server) {
		s.schemaRegistry = r
	}
}

func new(auth middleware.Authenticator, opts ...Option) *Server {
	s := &Server{
		auth:             auth,
		errorSensitivity: errawr.ErrorSensitivityNone,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func NewHandler(auth middleware.Authenticator, opts ...Option) http.Handler {
	s := new(auth, opts...)

	r := mux.NewRouter()
	s.Route(r)

	// These should run for every request, not just ones that Mux matches.
	var h http.Handler = r
	h = middleware.WebSecurity(h)
	h = utilapi.LogMiddleware(h)
	h = utilapi.RequestMiddleware(h)
	h = middleware.WithTrustedProxyHops(s.trustedProxyHops)(h)
	if s.capturer != nil {
		h = s.capturer.Middleware().Wrap(h)
	}

	return h
}
