package api

import (
	"fmt"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	"github.com/puppetlabs/nebula-tasks/pkg/manager/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type GetConditionsResponseEnvelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func (s *Server) GetConditions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	cm := managers.Conditions()

	cond, err := cm.Get(ctx)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	ev := evaluate.NewEvaluator(
		evaluate.WithSecretTypeResolver(resolve.NewSecretTypeResolver(managers.Secrets())),
		evaluate.WithOutputTypeResolver(resolve.NewOutputTypeResolver(managers.StepOutputs())),
		evaluate.WithAnswerTypeResolver(resolve.NewAnswerTypeResolver(managers.State())),
	)

	rv, rerr := ev.EvaluateAll(ctx, cond.Tree)
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(rerr.Error()).Bug())
		return
	}

	// Not being complete means there are unresolved "expressions" for this tree. These can include
	// parameters, outputs and secrets.
	if !rv.Complete() {
		uerr, ok := rv.Unresolvable.AsError().(*evaluate.UnresolvableError)
		if !ok {
			// This should never happen.
			utilapi.WriteError(ctx, w, errors.NewModelReadError().WithCause(uerr).Bug())
		}

		causes := make([]string, len(uerr.Causes))
		for i, cause := range uerr.Causes {
			causes[i] = cause.Error()
		}

		utilapi.WriteError(ctx, w, errors.NewExpressionUnresolvableError(causes))
		return
	}

	var failed bool

check:
	switch vt := rv.Value.(type) {
	case bool:
		failed = !vt
	case []interface{}:
		for _, cond := range vt {
			result, ok := cond.(bool)
			if !ok {
				utilapi.WriteError(ctx, w, errors.NewConditionTypeError(fmt.Sprintf("%T", vt)))
				return
			}

			if !result {
				failed = true
				break check
			}
		}
	default:
		utilapi.WriteError(ctx, w, errors.NewConditionTypeError(fmt.Sprintf("%T", vt)))
		return
	}

	var resp GetConditionsResponseEnvelope

	if failed {
		resp.Success = false
		resp.Message = "one or more conditions failed"
	} else {
		resp.Success = true
		resp.Message = "all checks passed"
	}

	utilapi.WriteObjectOK(ctx, w, resp)
}
