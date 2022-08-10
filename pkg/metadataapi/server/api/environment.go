package api

import (
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/relay-core/pkg/manager/specadapter"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/spec"
)

func (s *Server) GetEnvironmentVariable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	name, _ := middleware.Var(r, "name")

	environment, err := managers.Environment().Get(ctx)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	value, ok := environment.Value[name]
	if !ok {
		utilapi.WriteError(ctx, w, errors.NewModelNotFoundError())
		return
	}

	eval := spec.NewEvaluator(
		spec.WithConnectionTypeResolver{ConnectionTypeResolver: specadapter.NewConnectionTypeResolver(managers.Connections())},
		spec.WithSecretTypeResolver{SecretTypeResolver: specadapter.NewSecretTypeResolver(managers.Secrets())},
		spec.WithParameterTypeResolver{ParameterTypeResolver: specadapter.NewParameterTypeResolver(managers.Parameters())},
		spec.WithOutputTypeResolver{OutputTypeResolver: specadapter.NewOutputTypeResolver(managers.StepOutputs())},
		spec.WithStatusTypeResolver{StatusTypeResolver: specadapter.NewStatusTypeResolver(managers.ActionStatus())},
	)

	rv, rerr := evaluate.EvaluateAll(ctx, eval, value)
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(rerr.Error()))
		return
	}

	utilapi.WriteObjectOK(ctx, w, rv)
}

func (s *Server) GetEnvironment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	environment, err := managers.Environment().Get(ctx)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	eval := spec.NewEvaluator(
		spec.WithConnectionTypeResolver{ConnectionTypeResolver: specadapter.NewConnectionTypeResolver(managers.Connections())},
		spec.WithSecretTypeResolver{SecretTypeResolver: specadapter.NewSecretTypeResolver(managers.Secrets())},
		spec.WithParameterTypeResolver{ParameterTypeResolver: specadapter.NewParameterTypeResolver(managers.Parameters())},
		spec.WithOutputTypeResolver{OutputTypeResolver: specadapter.NewOutputTypeResolver(managers.StepOutputs())},
		spec.WithStatusTypeResolver{StatusTypeResolver: specadapter.NewStatusTypeResolver(managers.ActionStatus())},
	)

	rv, err := evaluate.EvaluateAll(ctx, eval, environment.Value)
	if err != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(err.Error()))
		return
	}

	utilapi.WriteObjectOK(ctx, w, NewGetSpecResponseEnvelope(rv))
}
