package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type Server struct {
	auth middleware.Authenticator
}

func (s *Server) Route(r *mux.Router) {
	r.UseEncodedPath()
	r.Use(middleware.WithAuthentication(s.auth))

	// Conditions
	r.HandleFunc("/conditions", s.GetConditions).Methods(http.MethodGet)

	// Events
	r.HandleFunc("/events", s.PostEvent).Methods(http.MethodPost)

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

func NewServer(auth middleware.Authenticator) *Server {
	return &Server{
		auth: auth,
	}
}

func NewHandler(auth middleware.Authenticator) http.Handler {
	r := mux.NewRouter()
	NewServer(auth).Route(r)
	return r
}
