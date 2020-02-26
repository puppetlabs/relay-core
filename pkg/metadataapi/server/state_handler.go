package server

import (
	"bytes"
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
)

type stateHandler struct {
	logger logging.Logger
}

func (st stateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st.get(w, r)
	case http.MethodPut:
		st.put(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (st stateHandler) get(w http.ResponseWriter, r *http.Request) {
	key, _ := shiftPath(r.URL.Path)

	managers := middleware.Managers(r)

	taskHash := middleware.TaskMetadata(r).Hash
	stm := managers.StateManager()

	ctx := r.Context()

	response, err := stm.Get(ctx, taskHash, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)
		return
	}

	ev, verr := transfer.EncodeJSON([]byte(response.Value))
	if verr != nil {
		utilapi.WriteError(ctx, w, errors.NewStateValueEncodingError().WithCause(verr).Bug())
		return
	}

	env := &state.StateEnvelope{
		Key:   response.Key,
		Value: ev,
	}

	utilapi.WriteObjectOK(ctx, w, env)
}

func (st stateHandler) put(w http.ResponseWriter, r *http.Request) {
	managers := middleware.Managers(r)
	md := middleware.TaskMetadata(r)

	stm := managers.StateManager()

	ctx := r.Context()

	buf := &bytes.Buffer{}
	buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err := stm.Set(ctx, md.Hash, buf); err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	w.WriteHeader(http.StatusCreated)
}
