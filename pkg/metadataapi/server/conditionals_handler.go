package server

import (
	"context"
	"encoding/hex"
	"fmt"
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
	md := middleware.TaskMetadata(r)

	taskId := hex.EncodeToString(md.Hash[:])
	h.logger.Info("handling condition request", "task-id", taskId)

	cm := managers.ConditionalsManager()

	conditionalsData, err := cm.GetByTaskID(ctx, taskId)
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
		evaluate.WithOutputTypeResolver(resolve.OutputTypeResolverFunc(func(ctx context.Context, from, name string) (interface{}, error) {
			o, err := managers.OutputsManager().Get(ctx, from, name)
			if errors.IsOutputsTaskNotFound(err) || errors.IsOutputsKeyNotFound(err) {
				return "", nil
			} else if err != nil {
				return "", err
			}

			return o.Value.Data, nil
		})),
		evaluate.WithAnswerTypeResolver(resolve.AnswerTypeResolverFunc(func(ctx context.Context, askRef, name string) (interface{}, error) {
			st, err := managers.StateManager().Get(ctx, md.Hash, name)
			if err != nil {
				return "", err
			}
			return st.Value, nil
		})),
	)

	rv, rerr := ev.EvaluateAll(ctx, tree)
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewTaskConditionEvaluationError().WithCause(rerr))

		return
	}

	// Not being complete means there are unresolved "expressions" for this tree. These can include
	// parameters, outputs and secrets. If there are unresolved secrets, then an UnsupportedConditionalExpressions
	// error is returned instead of an UnresolvedConditionalExpressions. This is because secrets are not supported
	// by the conditionals feature.
	if !rv.Complete() {
		var (
			err         errors.Error
			expressions []string
		)

		if len(rv.Unresolvable.Secrets) > 0 {
			for _, sec := range rv.Unresolvable.Secrets {
				expressions = append(expressions, "!Secret "+sec.Name)
			}

			err = errors.NewTaskUnsupportedConditionalExpressions(expressions)
		} else {
			if len(rv.Unresolvable.Outputs) > 0 {
				for _, o := range rv.Unresolvable.Outputs {
					expressions = append(expressions, fmt.Sprintf("!Output %s %s", o.Name, o.From))
				}
			}

			if len(rv.Unresolvable.Parameters) > 0 {
				for _, p := range rv.Unresolvable.Parameters {
					expressions = append(expressions, fmt.Sprintf("!Parameter %s", p.Name))
				}
			}

			if len(rv.Unresolvable.Answers) > 0 {
				for _, p := range rv.Unresolvable.Answers {
					expressions = append(expressions, fmt.Sprintf("!Answer %s %s", p.Name, p.AskRef))
				}
			}

			if len(rv.Unresolvable.Invocations) > 0 {
				for _, i := range rv.Unresolvable.Invocations {
					expressions = append(expressions, fmt.Sprintf("!Fn.%s (%s)", i.Name, i.Cause.Error()))
				}
			}

			err = errors.NewTaskUnresolvedConditionalExpressions(expressions)
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
