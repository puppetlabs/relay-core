package api

import (
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

type GetSecretResponseEnvelope struct {
	Key   string             `json:"key"`
	Value transfer.JSONOrStr `json:"value"`
}

func (s *Server) GetSecret(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	sm := managers.Secrets()

	name, _ := middleware.Var(r, "name")

	sec, err := sm.Get(ctx, name)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	ev, verr := transfer.EncodeJSON([]byte(sec.Value))
	if verr != nil {
		utilapi.WriteError(ctx, w, errors.NewModelReadError().WithCause(verr).Bug())
		return
	}

	env := &GetSecretResponseEnvelope{
		Key:   sec.Name,
		Value: ev,
	}

	utilapi.WriteObjectOK(ctx, w, env)
}
