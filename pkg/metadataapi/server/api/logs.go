package api

import (
	"encoding/json"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
)

func (s *Server) PostLogMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	var request plspb.LogMessageAppendRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
		return
	}

	response, err := managers.Logs().PostLogMessage(ctx, &request)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	utilapi.WriteObjectWithStatus(ctx, w, http.StatusAccepted, response)
}

func (s *Server) PostLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	var request plspb.LogCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
		return
	}

	response, err := managers.Logs().PostLog(ctx, &request)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	utilapi.WriteObjectCreated(ctx, w, response)
}
