package server

import (
	"context"
	"encoding/json"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/horsehead/logging"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type fetcherFunc func(context.Context, op.ManagerFactory, map[string]interface{}) (string, errors.Error)

var valueFetchers = map[task.SpecValueType]fetcherFunc{
	task.SpecValueSecret: fetchSecret,
	task.SpecValueOutput: fetchOutput,
}

type specsHandler struct {
	logger logging.Logger
}

func (h *specsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)

	var key string

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || "" != r.URL.Path {
		http.NotFound(w, r)

		return
	}

	h.logger.Info("handling spec request", "key", key)

	specData, err := managers.SpecsManager().GetByTaskID(ctx, key)
	if err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	var spec interface{}
	if err := json.Unmarshal([]byte(specData), &spec); nil != err {
		utilapi.WriteError(ctx, w, errors.NewTaskSpecDecodingError().WithCause(err))

		return
	}

	spec = h.expandValues(ctx, managers, spec)
	utilapi.WriteObjectOK(ctx, w, spec)
}

func (h *specsHandler) expandValues(ctx context.Context, managers op.ManagerFactory, spec interface{}) interface{} {
	switch v := spec.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))

		for index, elm := range v {
			result[index] = h.expandValues(ctx, managers, elm)
		}

		return result
	case map[string]interface{}:
		typ, ok := v["$type"].(string)
		if !ok {
			result := make(map[string]interface{})

			for key, val := range v {
				result[key] = h.expandValues(ctx, managers, val)
			}

			return result
		}

		valueType, ok := task.SpecValueMapping[typ]
		if !ok {
			h.logger.Warn("no such value type", "valueType", valueType)

			return ""
		}

		result, err := valueFetchers[valueType](ctx, managers, v)
		if err != nil {
			h.logger.Warn("failed to get value", "error", err, "spec", v)

			return ""
		}

		return result
	default:
		return v
	}
}

func fetchSecret(ctx context.Context, managers op.ManagerFactory, obj map[string]interface{}) (string, errors.Error) {
	name, ok := obj["name"].(string)
	if !ok {
		return "", errors.NewServerSecretFetcherNameValidationError()
	}

	secret, err := managers.SecretsManager().Get(ctx, name)
	if err != nil {
		return "", errors.NewServerSecretFetcherGetError().WithCause(err)
	}

	return secret.Value, nil
}

func fetchOutput(ctx context.Context, managers op.ManagerFactory, obj map[string]interface{}) (string, errors.Error) {
	name, ok := obj["name"].(string)
	if !ok {
		return "", errors.NewServerOutputFetcherNameValidationError()
	}

	taskName, ok := obj["taskName"].(string)
	if !ok {
		return "", errors.NewServerOutputFetcherTaskNameValidationError()
	}

	output, err := managers.OutputsManager().Get(ctx, taskName, name)
	if err != nil {
		return "", errors.NewServerOutputFetcherGetError().WithCause(err)
	}

	return output.Value, nil
}

func extractSecretName(obj map[string]interface{}) *string {
	if len(obj) != 2 {
		return nil
	}

	if ty, ok := obj["$type"].(string); !ok || "Secret" != ty {
		return nil
	}

	name, ok := obj["name"].(string)
	if !ok || "" == name {
		return nil
	}

	return &name
}
