package server

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/outputs"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type outputsHandler struct {
	logger logging.Logger
}

func (o outputsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		o.put(w, r)
	case http.MethodGet:
		o.get(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (o outputsHandler) put(w http.ResponseWriter, r *http.Request) {
	managers := middleware.Managers(r)
	md := middleware.TaskMetadata(r)

	om := managers.OutputsManager()

	ctx := r.Context()

	key, _ := shiftPath(r.URL.Path)

	var value transfer.JSONInterface

	switch r.Header.Get("Content-Type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(&value.Data); err != nil {
			utilapi.WriteError(ctx, w, errors.NewOutputsValueDecodingError().WithCause(err))
			return
		}
	case "text/plain", "application/octet-stream", "":
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(r.Body); err != nil {
			utilapi.WriteError(ctx, w, errors.NewOutputsValueDecodingError().WithCause(err))
			return
		}

		value.Data = buf.String()
	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := om.Put(ctx, md, key, value); err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (o outputsHandler) get(w http.ResponseWriter, r *http.Request) {
	var stepName string

	stepName, r.URL.Path = shiftPath(r.URL.Path)
	key, _ := shiftPath(r.URL.Path)

	managers := middleware.Managers(r)
	md := middleware.TaskMetadata(r)

	om := managers.OutputsManager()

	ctx := r.Context()

	response, err := om.Get(ctx, md, stepName, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)
		return
	}

	env := &outputs.Output{
		TaskName: response.TaskName,
		Key:      response.Key,
		Value:    response.Value,
	}

	utilapi.WriteObjectOK(ctx, w, env)
}
