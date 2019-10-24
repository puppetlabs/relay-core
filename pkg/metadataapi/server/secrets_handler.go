package server

import (
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

// Secret is the envelope type for a secret.
//
// TODO: Move to nebula-sdk?
type Secret struct {
	Key   string             `json:"key"`
	Value transfer.JSONOrStr `json:"value"`
}

type secretsHandler struct {
	logger logging.Logger
}

func (h *secretsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	sm := managers.SecretsManager()

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

	ev, verr := transfer.EncodeJSON([]byte(sec.Value))
	if verr != nil {
		utilapi.WriteError(ctx, w, errors.NewSecretsValueEncodingError().WithCause(verr).Bug())
		return
	}

	env := &Secret{
		Key:   sec.Key,
		Value: ev,
	}

	utilapi.WriteObjectOK(ctx, w, env)
}
