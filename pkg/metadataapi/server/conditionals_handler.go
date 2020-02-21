package server

import (
	"net/http"

	"github.com/kr/pretty"
	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type conditionalsHandler struct {
	logger logging.Logger
}

func (h *conditionalsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	var key string

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || r.URL.Path != "" {
		http.NotFound(w, r)

		return
	}

	h.logger.Info("handling condition request", "key", key)

	cm := managers.ConditionalsManager()

	conditionalsData, err := cm.GetByTaskID(ctx, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	tree, perr := parse.ParseJSONString(conditionalsData)
	if perr != nil {
		utilapi.WriteError(ctx, w, errors.NewTaskConditionalsDecodingError().WithCause(err))

		return
	}

	pretty.Println(tree)
}
