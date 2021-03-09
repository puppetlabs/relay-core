package server

import (
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
)

type GetHealthzResponseEnvelope struct {
	Ping string `json:"ping"`
}

func (*Server) GetHealthz(w http.ResponseWriter, r *http.Request) {
	utilapi.WriteObjectOK(r.Context(), w, &GetHealthzResponseEnvelope{Ping: "pong"})
}
