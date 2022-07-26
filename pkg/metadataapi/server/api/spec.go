package api

import (
	"net/http"

	"github.com/puppetlabs/leg/encoding/transfer"
	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/pathlang"
	"github.com/puppetlabs/leg/relspec/pkg/query"
	"github.com/puppetlabs/leg/relspec/pkg/ref"
	"github.com/puppetlabs/relay-core/pkg/manager/specadapter"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/spec"
)

type UnresolvableEnvelope struct {
	Data        []spec.DataID       `json:"data,omitempty"`
	Secrets     []spec.SecretID     `json:"secrets,omitempty"`
	Connections []spec.ConnectionID `json:"connections,omitempty"`
	Outputs     []spec.OutputID     `json:"outputs,omitempty"`
	Parameters  []spec.ParameterID  `json:"parameters,omitempty"`
	Answers     []spec.AnswerID     `json:"answers,omitempty"`
	Statuses    []spec.StatusID     `json:"statuses,omitempty"`
}

func appendUnresolvedOrErroredReferenceIDs[T ref.ID[T]](into *[]T) func(r ref.Reference[T]) {
	return func(r ref.Reference[T]) {
		if !r.Resolved() || r.Error() != nil {
			*into = append(*into, r.ID())
		}
	}
}

func NewUnresolvableEnvelope(refs *spec.References) *UnresolvableEnvelope {
	env := &UnresolvableEnvelope{}
	if refs != nil {
		refs.Data.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Data))
		refs.Secrets.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Secrets))
		refs.Connections.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Connections))
		refs.Outputs.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Outputs))
		refs.Parameters.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Parameters))
		refs.Answers.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Answers))
		refs.Statuses.ForEach(appendUnresolvedOrErroredReferenceIDs(&env.Statuses))
	}
	return env
}

type GetSpecResponseEnvelope struct {
	Value        transfer.JSONInterface `json:"value"`
	Unresolvable *UnresolvableEnvelope  `json:"unresolvable"`
	Complete     bool                   `json:"complete"`
}

func NewGetSpecResponseEnvelope(rv *evaluate.Result[*spec.References]) *GetSpecResponseEnvelope {
	return &GetSpecResponseEnvelope{
		Value:        transfer.JSONInterface{Data: rv.Value},
		Unresolvable: NewUnresolvableEnvelope(rv.References),
		Complete:     rv.OK(),
	}
}

func (s *Server) GetSpec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	data, err := managers.Spec().Get(ctx)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelReadError(err))
		return
	}

	lang := pathlang.New[*spec.References]().Expression
	switch r.URL.Query().Get("lang") {
	case "path-template":
		lang = pathlang.New[*spec.References]().Template
	case "jsonpath-template":
		lang = query.JSONPathTemplateLanguage[*spec.References]
	case "jsonpath":
		lang = query.JSONPathLanguage[*spec.References]
	case "", "path":
	default:
		utilapi.WriteError(ctx, w, errors.NewExpressionUnsupportedLanguageError(r.URL.Query().Get("lang")))
	}

	ev := spec.NewEvaluator(
		spec.WithConnectionTypeResolver{ConnectionTypeResolver: specadapter.NewConnectionTypeResolver(managers.Connections())},
		spec.WithSecretTypeResolver{SecretTypeResolver: specadapter.NewSecretTypeResolver(managers.Secrets())},
		spec.WithParameterTypeResolver{ParameterTypeResolver: specadapter.NewParameterTypeResolver(managers.Parameters())},
		spec.WithOutputTypeResolver{OutputTypeResolver: specadapter.NewOutputTypeResolver(managers.StepOutputs())},
		spec.WithStatusTypeResolver{StatusTypeResolver: specadapter.NewStatusTypeResolver(managers.ActionStatus())},
	)

	var rv *evaluate.Result[*spec.References]
	var rerr error
	if q := r.URL.Query().Get("q"); q != "" {
		rv, rerr = query.EvaluateQuery(ctx, ev, lang, data.Tree, q)
	} else {
		rv, rerr = evaluate.EvaluateAll(ctx, ev, data.Tree)
	}
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewExpressionEvaluationError(rerr.Error()))
		return
	}

	utilapi.WriteObjectOK(ctx, w, NewGetSpecResponseEnvelope(rv))
}
