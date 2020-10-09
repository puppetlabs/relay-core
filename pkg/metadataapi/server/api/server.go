package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/workflow/spec"
)

type ServerOption func(*Server)

func WithSpecSchemaRegistry(reg spec.SchemaRegistry) ServerOption {
	return func(s *Server) {
		s.specSchemaRegistry = reg
	}
}

type Server struct {
	auth               middleware.Authenticator
	specSchemaRegistry spec.SchemaRegistry
}

func (s *Server) Route(r *mux.Router) {
	r.UseEncodedPath()
	r.Use(middleware.WithAuthentication(s.auth))

	// Conditions
	r.HandleFunc("/conditions", s.GetConditions).Methods(http.MethodGet)

	// Events
	r.HandleFunc("/events", s.PostEvent).Methods(http.MethodPost)

	// Environment
	r.HandleFunc("/environment", s.GetEnvironment).Methods(http.MethodGet)
	r.HandleFunc("/environment/{name}", s.GetEnvironmentVariable).Methods(http.MethodGet)

	// Outputs
	r.HandleFunc("/outputs/{name}", s.PutOutput).Methods(http.MethodPut)
	r.HandleFunc("/outputs/{stepName}/{name}", s.GetOutput).Methods(http.MethodGet)

	// Secrets
	r.HandleFunc("/secrets/{name}", s.GetSecret).Methods(http.MethodGet)

	// Spec
	r.HandleFunc("/spec", s.GetSpec).Methods(http.MethodGet)

	// State
	r.HandleFunc("/state/{name}", s.GetState).Methods(http.MethodGet)

	// Validation
	r.HandleFunc("/validate", s.PostValidate).Methods(http.MethodPost)
}

func NewServer(auth middleware.Authenticator, opts ...ServerOption) *Server {
	svr := &Server{
		auth: auth,
	}

	for _, opt := range opts {
		opt(svr)
	}

	return svr
}

func NewHandler(auth middleware.Authenticator, opts ...ServerOption) http.Handler {
	r := mux.NewRouter()
	NewServer(auth, opts...).Route(r)

	return r
}
