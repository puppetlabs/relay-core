package server

import (
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
)

type secretsHandler struct {
	managers op.ManagerFactory
}

func (h *secretsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sm, err := h.managers.SecretsManager()
	if err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	if err := sm.Login(ctx); err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	var key string

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || "" != r.URL.Path {
		http.NotFound(w, r)

		return
	}

	sec, err := sm.Get(ctx, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	utilapi.WriteObjectOK(ctx, w, sec)
}
