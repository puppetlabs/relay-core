package server

import (
	"context"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/conditionals"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type conditionalsHandler struct {
	logger logging.Logger
}

func (h *conditionalsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)

		return
	}

	h.get(w, r)
}

func (h *conditionalsHandler) get(w http.ResponseWriter, r *http.Request) {
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

	ev := evaluate.NewEvaluator(
		evaluate.WithOutputTypeResolver(resolve.OutputTypeResolverFunc(func(ctx context.Context, from, name string) (string, error) {
			o, err := managers.OutputsManager().Get(ctx, from, name)
			if errors.IsOutputsTaskNotFound(err) || errors.IsOutputsKeyNotFound(err) {
				return "", nil
			} else if err != nil {
				return "", err
			}

			return o.Value, nil
		})),
	)

	rv, rerr := ev.EvaluateAll(ctx, tree)
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewTaskConditionEvaluationError().WithCause(rerr))

		return
	}

	if !rv.Complete() {
		var err errors.Error

		if len(rv.Unresolvable.Secrets) > 0 {
			expressions := []string{}

			for _, sec := range rv.Unresolvable.Secrets {
				expressions = append(expressions, "!Secret "+sec.Name)
			}

			err = errors.NewTaskUnsupportedConditionalExpressions(expressions)
		} else {
			err = errors.NewTaskConditionUnresolvedError()
		}

		utilapi.WriteError(ctx, w, err)

		return
	}

	conditions, ok := rv.Value.(map[string]interface{})["conditions"].([]interface{})
	if !ok {
		utilapi.WriteError(ctx, w, errors.NewTaskConditionStructureMalformedError())

		return
	}

	var failed bool

	for _, cond := range conditions {
		result, ok := cond.(bool)
		if !ok {
			utilapi.WriteError(ctx, w, errors.NewTaskConditionStructureMalformedError())

			return
		}

		if !result {
			failed = true

			break
		}
	}

	var resp conditionals.ResponseEnvelope

	if failed {
		resp.Success = false
		resp.Message = "one or more conditions failed"
	} else {
		resp.Success = true
		resp.Message = "all checks passed"
	}

	utilapi.WriteObjectOK(ctx, w, resp)
}
