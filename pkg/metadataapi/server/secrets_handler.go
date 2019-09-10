package server

import (
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/horsehead/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
)

type secretsHandler struct {
	managers op.ManagerFactory
	logger   logging.Logger
}

func (h *secretsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sm := h.managers.SecretsManager()

	var key string

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || "" != r.URL.Path {
		http.NotFound(w, r)

		return
	}

	h.logger.Info("handling secret request", "key", key)

	sec, err := sm.Get(ctx, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	utilapi.WriteObjectOK(ctx, w, sec)
}
