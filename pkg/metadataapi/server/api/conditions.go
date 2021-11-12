package api

import (
	"fmt"
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/manager/resolve"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

type GetConditionsResponseEnvelope struct {
	Resolved bool   `json:"resolved"`
	Success  bool   `json:"success"`
	Message  string `json:"message"`
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
		evaluate.WithConnectionTypeResolver{ConnectionTypeResolver: resolve.NewConnectionTypeResolver(managers.Connections())},
		evaluate.WithSecretTypeResolver{SecretTypeResolver: resolve.NewSecretTypeResolver(managers.Secrets())},
		evaluate.WithParameterTypeResolver{ParameterTypeResolver: resolve.NewParameterTypeResolver(managers.Parameters())},
		evaluate.WithOutputTypeResolver{OutputTypeResolver: resolve.NewOutputTypeResolver(managers.StepOutputs())},
		evaluate.WithAnswerTypeResolver{AnswerTypeResolver: resolve.NewAnswerTypeResolver(managers.State())},
	)

	rv, rerr := model.EvaluateAll(ctx, ev, cond.Tree)
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(rerr.Error()))
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
				if rv.Complete() {
					utilapi.WriteError(ctx, w, errors.NewConditionTypeError(fmt.Sprintf("%T", cond)))
					return
				}
				continue
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

	resp := GetConditionsResponseEnvelope{
		Resolved: rv.Complete(),
	}

	if failed {
		resp.Success = false
		resp.Message = "one or more conditions failed"
		utilapi.WriteObjectOK(ctx, w, resp)
		return
	}

	// Not being complete means there are unresolved "expressions" for this tree. These can include
	// parameters, outputs, secrets, etc.
	if !rv.Complete() {
		uerr, ok := rv.Unresolvable.AsError().(*model.UnresolvableError)
		if !ok {
			// This should never happen.
			utilapi.WriteError(ctx, w, errors.NewModelReadError().WithCause(uerr).Bug())
		}

		causes := make([]string, len(uerr.Causes))
		for i, cause := range uerr.Causes {
			causes[i] = cause.Error()
		}

		resp.Message = errors.NewExpressionUnresolvableError(causes).Error()
		utilapi.WriteObjectOK(ctx, w, resp)
		return
	}

	resp.Success = true
	resp.Message = "all checks passed"

	utilapi.WriteObjectOK(ctx, w, resp)
}
