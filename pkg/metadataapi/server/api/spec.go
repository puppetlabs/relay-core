package api

import (
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/query"
	"github.com/puppetlabs/relay-core/pkg/manager/resolve"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

func (s *Server) GetSpec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	spec, err := managers.Spec().Get(ctx)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	lang := query.PathLanguage()
	switch r.URL.Query().Get("lang") {
	case "path-template":
		lang = query.PathTemplateLanguage()
	case "jsonpath-template":
		lang = query.JSONPathTemplateLanguage
	case "jsonpath":
		lang = query.JSONPathLanguage
	case "", "path":
	default:
		utilapi.WriteError(ctx, w, errors.NewExpressionUnsupportedLanguageError(r.URL.Query().Get("lang")))
	}

	ev := evaluate.NewEvaluator(
		evaluate.WithConnectionTypeResolver{ConnectionTypeResolver: resolve.NewConnectionTypeResolver(managers.Connections())},
		evaluate.WithSecretTypeResolver{SecretTypeResolver: resolve.NewSecretTypeResolver(managers.Secrets())},
		evaluate.WithParameterTypeResolver{ParameterTypeResolver: resolve.NewParameterTypeResolver(managers.Parameters())},
		evaluate.WithOutputTypeResolver{OutputTypeResolver: resolve.NewOutputTypeResolver(managers.StepOutputs())},
	)

	var rv *model.Result
	var rerr error
	if q := r.URL.Query().Get("q"); q != "" {
		rv, rerr = query.EvaluateQuery(ctx, ev, lang, spec.Tree, q)
	} else {
		rv, rerr = model.EvaluateAll(ctx, ev, spec.Tree)
	}
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(rerr.Error()))
		return
	}

	utilapi.WriteObjectOK(ctx, w, model.NewJSONResultEnvelope(rv))
}
