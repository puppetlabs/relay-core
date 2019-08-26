package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type specsHandler struct {
	managers  op.ManagerFactory
	namespace string
}

func (h *specsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := rest.InClusterConfig()
	if nil != err {
		utilapi.WriteError(
			ctx,
			w,
			errors.NewServerInClusterConfigError().WithCause(err))
		return
	}
	client, err := kubernetes.NewForConfig(config)
	if nil != err {
		utilapi.WriteError(
			ctx,
			w,
			errors.NewServerNewK8sClientError().WithCause(err))
		return
	}
	var key string

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || "" != r.URL.Path {
		http.NotFound(w, r)

		return
	}

	configMap, err := client.CoreV1().ConfigMaps(h.namespace).Get(key, metav1.GetOptions{})
	if nil != err {
		utilapi.WriteError(
			ctx,
			w,
			errors.NewServerGetConfigMapError(key, h.namespace).WithCause(err))
		return
	}
	var spec interface{}
	if err := json.Unmarshal([]byte(configMap.Data["spec.json"]), &spec); nil != err {
		utilapi.WriteError(
			ctx,
			w,
			errors.NewServerConfigMapJSONError(key, h.namespace).WithCause(err))
		return
	}

	sm, merr := h.managers.SecretsManager()
	if merr != nil {
		utilapi.WriteError(ctx, w, merr)

		return
	}

	if err := sm.Login(ctx); err != nil {
		utilapi.WriteError(ctx, w, err)

		return
	}

	spec = h.expandSecrets(ctx, sm, spec)
	utilapi.WriteObjectOK(ctx, w, spec)
}

func (h *specsHandler) expandSecrets(ctx context.Context, sm op.SecretsManager, spec interface{}) interface{} {
	switch v := spec.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for index, elm := range v {
			result[index] = h.expandSecrets(ctx, sm, elm)
		}
		return result
	case map[string]interface{}:
		secretName := extractSecretName(v)
		if nil != secretName {
			sec, err := sm.Get(ctx, *secretName)
			if err != nil || nil == sec {
				log.Printf("failed to get secret=%s: %v", *secretName, err)
				return ""
			}
			return sec.Value
		}
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = h.expandSecrets(ctx, sm, val)
		}
		return result
	default:
		return v
	}

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
