package api

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

type GetOutputResponseEnvelope struct {
	TaskName string                 `json:"task_name"`
	Key      string                 `json:"key"`
	Value    transfer.JSONInterface `json:"value"`
}

func (s *Server) GetOutput(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	om := managers.StepOutputs()

	stepName, _ := middleware.Var(r, "stepName")
	name, _ := middleware.Var(r, "name")

	output, err := om.Get(ctx, stepName, name)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	env := &GetOutputResponseEnvelope{
		TaskName: output.Step.Name,
		Key:      output.Name,
		Value:    transfer.JSONInterface{Data: output.Value},
	}

	utilapi.WriteObjectOK(ctx, w, env)
}

func (s *Server) PutOutput(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	om := managers.StepOutputs()

	name, _ := middleware.Var(r, "name")

	var value transfer.JSONInterface

	switch r.Header.Get("content-type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(&value.Data); err != nil {
			utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
			return
		}
	case "text/plain", "application/octet-stream", "":
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(r.Body); err != nil {
			utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
			return
		}

		value.Data = buf.String()
	default:
		utilapi.WriteError(ctx, w, errors.NewAPIUnknownRequestMediaTypeError(r.Header.Get("content-type")))
		return
	}

	if _, err := om.Set(ctx, name, value.Data); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
