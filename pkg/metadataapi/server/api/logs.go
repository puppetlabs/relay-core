package api

import (
	"bytes"
	"context"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

func (s *Server) PostLogMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	id, _ := middleware.Var(r, "logId")

	switch r.Header.Get("content-type") {
	case "application/octet-stream":
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(r.Body); err != nil {
			utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
			return
		}

		response, err := managers.Logs().PostLogMessage(ctx, id, buf.String())
		if err != nil {
			utilapi.WriteError(ctx, w, ModelWriteError(err))
			return
		}

		WriteObjectWithStatus(ctx, w, http.StatusAccepted, response)
		return
	default:
		utilapi.WriteError(ctx, w, errors.NewAPIUnknownRequestMediaTypeError(r.Header.Get("content-type")))
		return
	}
}

func (s *Server) PostLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	switch r.Header.Get("content-type") {
	case "application/octet-stream":
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(r.Body); err != nil {
			utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
			return
		}

		response, err := managers.Logs().PostLog(ctx, buf.String())
		if err != nil {
			utilapi.WriteError(ctx, w, ModelWriteError(err))
			return
		}

		WriteObjectWithStatus(ctx, w, http.StatusCreated, response)
		return
	default:
		utilapi.WriteError(ctx, w, errors.NewAPIUnknownRequestMediaTypeError(r.Header.Get("content-type")))
		return
	}
}

func WriteObjectWithStatus(ctx context.Context, w http.ResponseWriter, status int, b []byte) {
	w.Header().Set("content-type", "application/octet-stream")

	w.WriteHeader(status)

	if _, err := w.Write(b); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
	}
}
