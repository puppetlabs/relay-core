package server

import (
	"bytes"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/horsehead/logging"
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

	taskName := md.Name
	key, _ := shiftPath(r.URL.Path)

	buf := &bytes.Buffer{}
	buf.ReadFrom(r.Body)
	defer r.Body.Close()

	if err := om.Put(ctx, taskName, key, buf); err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (o outputsHandler) get(w http.ResponseWriter, r *http.Request) {
	var taskName string

	taskName, r.URL.Path = shiftPath(r.URL.Path)
	key, _ := shiftPath(r.URL.Path)

	managers := middleware.Managers(r)

	om := managers.OutputsManager()

	ctx := r.Context()

	response, err := om.Get(ctx, taskName, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	utilapi.WriteObjectOK(ctx, w, response)
}
