package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/workflow/spec"
)

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
}

func NewServer(auth middleware.Authenticator, specSchemaRegistry spec.SchemaRegistry) *Server {
	return &Server{
		auth:               auth,
		specSchemaRegistry: specSchemaRegistry,
	}
}

func NewHandler(auth middleware.Authenticator, specSchemaRegistry spec.SchemaRegistry) http.Handler {
	r := mux.NewRouter()
	NewServer(auth, specSchemaRegistry).Route(r)
	return r
}
