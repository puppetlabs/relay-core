package api

import (
	"encoding/json"
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

func (s *Server) PostDecorator(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	dm := managers.StepDecorators()

	name, _ := middleware.Var(r, "name")

	var values = make(map[string]interface{})

	if err := json.NewDecoder(r.Body).Decode(&values); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))

		return
	}

	typ, ok := values["type"].(string)
	if !ok {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError())

		return
	}

	if err := dm.Set(ctx, typ, name, values); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))

		return
	}

	w.WriteHeader(http.StatusCreated)
}
