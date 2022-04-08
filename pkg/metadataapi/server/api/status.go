package api

import (
	"encoding/json"
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type PutActionStatusRequestEnvelope struct {
	ExitCode int `json:"exitCode"`
}

func (s *Server) PutActionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	asm := managers.ActionStatus()

	var env PutActionStatusRequestEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
		return
	}

	ActionStatus := &model.ActionStatus{
		ExitCode: env.ExitCode,
	}

	if err := asm.Set(ctx, ActionStatus); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
