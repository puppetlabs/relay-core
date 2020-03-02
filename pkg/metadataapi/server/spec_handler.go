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

type specHandler struct {
	logger logging.Logger
}

func (h *specHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	m := middleware.Managers(r)
	md := middleware.TaskMetadata(r)

	h.logger.Info("handling spec request", "task-id", md.Hash.HexEncoding())

	spec, err := m.SpecsManager().Get(ctx, md.Hash)
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

	utilapi.WriteObjectOK(ctx, w, evaluate.NewJSONResultEnvelope(rv))
}
