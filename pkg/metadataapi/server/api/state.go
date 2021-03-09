package api

import (
	"net/http"

	"github.com/puppetlabs/leg/encoding/transfer"
	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

type GetStateResponseEnvelope struct {
	Key   string                 `json:"key"`
	Value transfer.JSONInterface `json:"value"`
}

func (s *Server) GetState(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	sm := managers.State()

	name, _ := middleware.Var(r, "name")

	state, err := sm.Get(ctx, name)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	env := &GetStateResponseEnvelope{
		Key:   state.Name,
		Value: transfer.JSONInterface{Data: state.Value},
	}

	utilapi.WriteObjectOK(ctx, w, env)
}
