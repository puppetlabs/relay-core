package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/api"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type Server struct {
	auth             middleware.Authenticator
	errorSensitivity errawr.ErrorSensitivity
	trustedProxyHops int
}

func (s *Server) Route(r *mux.Router) {
	r.Use(middleware.WithTrustedProxyHops(s.trustedProxyHops))
	r.Use(utilapi.RequestMiddleware)
	r.Use(utilapi.LogMiddleware)
	r.Use(middleware.WithErrorSensitivity(s.errorSensitivity))

	r.HandleFunc("/healthz", s.GetHealthz).Methods("GET")

	// This has a different set of middleware so bind it under a subrouter.
	api.NewServer(s.auth).Route(r.NewRoute().Subrouter())
}

type Option func(s *Server)

func WithErrorSensitivity(sensitivity errawr.ErrorSensitivity) Option {
	return func(s *Server) {
		s.errorSensitivity = sensitivity
	}
}

func WithTrustedProxyHops(n int) Option {
	return func(s *Server) {
		s.trustedProxyHops = n
	}
}

func New(auth middleware.Authenticator, opts ...Option) *Server {
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
	r := mux.NewRouter()
	New(auth, opts...).Route(r)
	return r
}
