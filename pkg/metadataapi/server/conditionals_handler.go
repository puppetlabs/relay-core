package server

import (
	"context"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
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

	h.logger.Info("handling condition request", "task-id", md.Hash.HexEncoding())

	cm := managers.ConditionalsManager()

	tree, err := cm.Get(ctx, md)
	if err != nil {
		utilapi.WriteError(ctx, w, err)
		return
	}

	ev := evaluate.NewEvaluator(
		evaluate.WithOutputTypeResolver(resolve.OutputTypeResolverFunc(func(ctx context.Context, from, name string) (interface{}, error) {
			o, err := managers.OutputsManager().Get(ctx, md, from, name)
			if errors.IsOutputsTaskNotFound(err) || errors.IsOutputsKeyNotFound(err) {
				return nil, &resolve.OutputNotFoundError{From: from, Name: name}
			} else if err != nil {
				return nil, err
			}

			return o.Value.Data, nil
		})),
		evaluate.WithAnswerTypeResolver(resolve.AnswerTypeResolverFunc(func(ctx context.Context, askRef, name string) (interface{}, error) {
			st, err := managers.StateManager().Get(ctx, md, name)
			if errors.IsStateTaskNotFound(err) || errors.IsStateNotFoundForID(err) || errors.IsStateKeyNotFound(err) {
				return nil, &resolve.AnswerNotFoundError{AskRef: askRef, Name: name}
			} else if err != nil {
				return nil, err
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
		} else if uerr, ok := rv.Unresolvable.AsError().(*evaluate.UnresolvableError); ok {
			for _, cause := range uerr.Causes {
				expressions = append(expressions, cause.Error())
			}

			err = errors.NewTaskUnresolvedConditionalExpressions(expressions)
		} else {
			err = errors.NewTaskConditionEvaluationError().WithCause(rv.Unresolvable.AsError()).Bug()
		}

		utilapi.WriteError(ctx, w, err)
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
				utilapi.WriteError(ctx, w, errors.NewTaskConditionStructureMalformedError())
				return
			}

			if !result {
				failed = true
				break check
			}
		}
	default:
		utilapi.WriteError(ctx, w, errors.NewTaskConditionStructureMalformedError())
		return
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
