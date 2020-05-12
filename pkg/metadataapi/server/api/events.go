package api

import (
	"encoding/json"
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type PostEventRequestEnvelope struct {
	Data map[string]transfer.JSONInterface `json:"data"`
}

func (s *Server) PostEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	em := managers.Events()

	var env PostEventRequestEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
		return
	}

	data := make(map[string]interface{}, len(env.Data))
	for k, v := range env.Data {
		data[k] = v.Data
	}

	if _, err := em.Emit(ctx, data); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
