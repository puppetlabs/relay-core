package api

import (
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	"github.com/puppetlabs/nebula-tasks/pkg/manager/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
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

	utilapi.WriteObjectOK(ctx, w, evaluate.NewJSONResultEnvelope(rv))
}
