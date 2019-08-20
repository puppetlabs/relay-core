package server

import (
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
)

type secretsHandler struct {
	sec secrets.Store
}

func (h *secretsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var key string

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || "" != r.URL.Path {
		http.NotFound(w, r)

		return
	}

	sec, err := h.sec.Get(r.Context(), key)
	if err != nil {
		utilapi.WriteError(r.Context(), w, err)

		return
	}

	utilapi.WriteObjectOK(r.Context(), w, sec)
}
