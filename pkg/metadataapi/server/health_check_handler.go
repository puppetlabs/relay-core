package server

import (
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
)

type status struct {
	Message string `json:"message"`
}

type healthCheckHandler struct {
}

func (h healthCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	utilapi.WriteObjectOK(r.Context(), w, status{Message: "ready"})
}
