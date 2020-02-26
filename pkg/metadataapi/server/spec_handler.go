package server

import (
	"context"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/puppetlabs/horsehead/v2/logging"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
)

type UnresolvableSecretEnvelope struct {
	Name string `json:"name"`
}

type UnresolvableOutputEnvelope struct {
	From string `json:"from"`
	Name string `json:"name"`
}

type UnresolvableParameterEnvelope struct {
	Name string `json:"name"`
}

type UnresolvableInvocationEnvelope struct {
	Name string `json:"name"`
}

type UnresolvableEnvelope struct {
	Secrets     []*UnresolvableSecretEnvelope     `json:"secrets,omitempty"`
	Outputs     []*UnresolvableOutputEnvelope     `json:"outputs,omitempty"`
	Parameters  []*UnresolvableParameterEnvelope  `json:"parameters,omitempty"`
	Invocations []*UnresolvableInvocationEnvelope `json:"invocations,omitempty"`
}

func NewUnresolvableEnvelope(ur evaluate.Unresolvable) *UnresolvableEnvelope {
	env := &UnresolvableEnvelope{}

	if len(ur.Secrets) > 0 {
		env.Secrets = make([]*UnresolvableSecretEnvelope, len(ur.Secrets))
		for i, s := range ur.Secrets {
			env.Secrets[i] = &UnresolvableSecretEnvelope{
				Name: s.Name,
			}
		}
	}

	if len(ur.Outputs) > 0 {
		env.Outputs = make([]*UnresolvableOutputEnvelope, len(ur.Outputs))
		for i, o := range ur.Outputs {
			env.Outputs[i] = &UnresolvableOutputEnvelope{
				From: o.From,
				Name: o.Name,
			}
		}
	}

	if len(ur.Parameters) > 0 {
		env.Parameters = make([]*UnresolvableParameterEnvelope, len(ur.Parameters))
		for i, p := range ur.Parameters {
			env.Parameters[i] = &UnresolvableParameterEnvelope{
				Name: p.Name,
			}
		}
	}

	if len(ur.Invocations) > 0 {
		env.Invocations = make([]*UnresolvableInvocationEnvelope, len(ur.Invocations))
		for i, call := range ur.Invocations {
			// TODO: Add cause?
			env.Invocations[i] = &UnresolvableInvocationEnvelope{
				Name: call.Name,
			}
		}
	}

	return env
}

type ResultEnvelope struct {
	Value        interface{}           `json:"value"`
	Unresolvable *UnresolvableEnvelope `json:"unresolvable"`
	Complete     bool                  `json:"complete"`
}

func NewResultEnvelope(rv *evaluate.Result) *ResultEnvelope {
	return &ResultEnvelope{
		Value:        rv.Value,
		Unresolvable: NewUnresolvableEnvelope(rv.Unresolvable),
		Complete:     rv.Complete(),
	}
}

type specHandler struct {
	logger logging.Logger
}

func (h *specHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	m := middleware.Managers(r)
	md := middleware.TaskMetadata(r)

	h.logger.Info("handling spec request", "task-name", md.Name)

	spec, err := m.SpecsManager().GetByTaskID(ctx, md.Name)
	if err != nil {
		utilapi.WriteError(ctx, w, err)
		return
	}

	tree, perr := parse.ParseJSONString(spec)
	if perr != nil {
		utilapi.WriteError(ctx, w, errors.NewTaskSpecDecodingError().WithCause(perr))
		return
	}

	ev := evaluate.NewEvaluator(
		evaluate.WithSecretTypeResolver(resolve.SecretTypeResolverFunc(func(ctx context.Context, name string) (string, error) {
			s, err := m.SecretsManager().Get(ctx, name)
			if errors.IsSecretsKeyNotFound(err) {
				return "", &resolve.SecretNotFoundError{Name: name}
			} else if err != nil {
				return "", err
			}
			return s.Value, nil
		})),
		evaluate.WithOutputTypeResolver(resolve.OutputTypeResolverFunc(func(ctx context.Context, from, name string) (interface{}, error) {
			o, err := m.OutputsManager().Get(ctx, from, name)
			if errors.IsOutputsTaskNotFound(err) || errors.IsOutputsKeyNotFound(err) {
				return nil, &resolve.OutputNotFoundError{From: from, Name: name}
			} else if err != nil {
				return nil, err
			}
			return o.Value.Data, nil
		})),
		evaluate.WithResultMapper(evaluate.NewUTF8SafeResultMapper()),
	)

	var rv *evaluate.Result
	var rerr error
	if query := r.URL.Query().Get("q"); query != "" {
		rv, rerr = ev.EvaluateQuery(ctx, tree, query)
	} else {
		rv, rerr = ev.EvaluateAll(ctx, tree)
	}
	if rerr != nil {
		utilapi.WriteError(ctx, w, errors.NewTaskSpecEvaluationError().WithCause(rerr))
		return
	}

	utilapi.WriteObjectOK(ctx, w, NewResultEnvelope(rv))
}
