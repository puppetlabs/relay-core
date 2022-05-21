package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/relay-core/pkg/manager/specadapter"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/spec"
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

	condition, err := cm.Get(ctx)
	if err != nil && err != model.ErrNotFound {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	if condition == nil || condition.Tree == nil {
		utilapi.WriteObjectOK(ctx, w,
			GetConditionsResponseEnvelope{
				Resolved: true,
				Success:  true,
			})
		return
	}

	ev := spec.NewEvaluator(
		spec.WithConnectionTypeResolver{ConnectionTypeResolver: specadapter.NewConnectionTypeResolver(managers.Connections())},
		spec.WithSecretTypeResolver{SecretTypeResolver: specadapter.NewSecretTypeResolver(managers.Secrets())},
		spec.WithParameterTypeResolver{ParameterTypeResolver: specadapter.NewParameterTypeResolver(managers.Parameters())},
		spec.WithOutputTypeResolver{OutputTypeResolver: specadapter.NewOutputTypeResolver(managers.StepOutputs())},
		spec.WithAnswerTypeResolver{AnswerTypeResolver: specadapter.NewAnswerTypeResolver(managers.State())},
		spec.WithStatusTypeResolver{StatusTypeResolver: specadapter.NewStatusTypeResolver(managers.ActionStatus())},
	)

	rv, err := evaluate.EvaluateAll(ctx, ev, condition.Tree)
	if err != nil {
		message := err.Error()
		addStepMessage(r, message,
			&model.ConditionEvaluationResult{
				Expression: condition.Tree,
			}, nil)
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(message))
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
				if rv.OK() {
					err := errors.NewConditionTypeError(fmt.Sprintf("%T", cond))
					addStepMessage(r, err.Error(),
						&model.ConditionEvaluationResult{
							Expression: condition.Tree,
						}, nil)
					utilapi.WriteError(ctx, w, err)
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
		if rv.OK() {
			err := errors.NewConditionTypeError(fmt.Sprintf("%T", vt))
			addStepMessage(r, err.Error(),
				&model.ConditionEvaluationResult{
					Expression: condition.Tree,
				}, nil)
			utilapi.WriteError(ctx, w, err)
			return
		}
	}

	resp := GetConditionsResponseEnvelope{
		Resolved: rv.OK(),
	}

	if failed {
		// Override the resolved flag if the condition failed.
		resp.Resolved = true
		resp.Success = false
		resp.Message = "one or more conditions failed"
		utilapi.WriteObjectOK(ctx, w, resp)
		return
	}

	// Not being complete means there are unresolved "expressions" for this tree. These can include
	// parameters, outputs, secrets, etc.
	if uerr := rv.References.ToError(); uerr != nil {
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

func addStepMessage(r *http.Request, message string,
	conditionEvaluationResult *model.ConditionEvaluationResult,
	schemaValidationResult *model.SchemaValidationResult) {

	ctx := r.Context()

	managers := middleware.Managers(r)
	sm := managers.StepMessages()

	// TODO This needs better handling
	details := message
	if len(details) > 1024 {
		details = details[:1024]
	}

	stepMessage := &model.StepMessage{
		ID:                        uuid.NewString(),
		Details:                   details,
		Time:                      time.Now(),
		ConditionEvaluationResult: conditionEvaluationResult,
		SchemaValidationResult:    schemaValidationResult,
	}

	_ = sm.Set(ctx, stepMessage)
}
