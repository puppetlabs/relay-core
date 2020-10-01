package api

import (
	goerrors "errors"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/manager/resolve"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/util/image"
	wfspec "github.com/puppetlabs/relay-core/pkg/workflow/spec"
)

func (s *Server) GetSpec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	spec, err := managers.Spec().Get(ctx)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	lang := evaluate.LanguagePath
	switch r.URL.Query().Get("lang") {
	case "jsonpath-template":
		lang = evaluate.LanguageJSONPathTemplate
	case "jsonpath":
		lang = evaluate.LanguageJSONPath
	case "", "path":
	default:
		utilapi.WriteError(ctx, w, errors.NewExpressionUnsupportedLanguageError(r.URL.Query().Get("lang")))
	}

	ev := evaluate.NewEvaluator(
		evaluate.WithLanguage(lang),
		evaluate.WithConnectionTypeResolver(resolve.NewConnectionTypeResolver(managers.Connections())),
		evaluate.WithParameterTypeResolver(resolve.NewParameterTypeResolver(managers.Parameters())),
		evaluate.WithOutputTypeResolver(resolve.NewOutputTypeResolver(managers.StepOutputs())),
		evaluate.WithSecretTypeResolver(resolve.NewSecretTypeResolver(managers.Secrets())),
	).ScopeTo(spec.Tree)

	var rv *evaluate.Result
	var rerr error
	if query := r.URL.Query().Get("q"); query != "" {
		rv, rerr = ev.EvaluateQuery(ctx, query)
	} else {
		rv, rerr = ev.EvaluateAll(ctx)
	}
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(rerr.Error()))
		return
	}

	env := evaluate.NewJSONResultEnvelope(rv)

	if s.specSchemaRegistry != nil && env.Complete {
		sm, err := managers.StepMetadata().Get(ctx)
		if err != nil {
			utilapi.WriteError(ctx, w, ModelReadError(err))
			return
		}

		ref, err := image.RepoReference(sm.Image)
		if err != nil {
			utilapi.WriteError(ctx, w, errors.NewStepImageParseError().WithCause(err))
			return
		}

		repository := ref.Context()

		schema, err := s.specSchemaRegistry.GetByStepRepository(repository.RepositoryStr())
		if err != nil {
			if !goerrors.Is(err, &wfspec.SchemaDoesNotExistError{}) {
				utilapi.WriteError(ctx, w, errors.NewSpecSchemaLookupError().WithCause(err))

				return
			}
		} else {
			if err := schema.ValidateGo(env.Value.Data); err != nil {
				utilapi.WriteError(ctx, w, errors.NewSpecSchemaValidationError().WithCause(err))

				return
			}
		}
	}

	utilapi.WriteObjectOK(ctx, w, env)
}
