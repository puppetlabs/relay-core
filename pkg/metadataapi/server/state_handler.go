package server

import (
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type stateHandler struct {
	logger logging.Logger
}

func (st stateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st.get(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (st stateHandler) get(w http.ResponseWriter, r *http.Request) {
	var stepName string

	stepName, r.URL.Path = shiftPath(r.URL.Path)
	key, _ := shiftPath(r.URL.Path)

	managers := middleware.Managers(r)

	stm := managers.StateManager()

	ctx := r.Context()

	response, err := stm.Get(ctx, stepName, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)
		return
	}

	ev, verr := transfer.EncodeJSON([]byte(response.Value))
	if verr != nil {
		utilapi.WriteError(ctx, w, errors.NewOutputsValueEncodingError().WithCause(verr).Bug())
		return
	}

	env := &outputs.Output{
		TaskName: response.TaskName,
		Key:      response.Key,
		Value:    ev,
	}

	utilapi.WriteObjectOK(ctx, w, env)
}
