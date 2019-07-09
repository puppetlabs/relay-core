package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type specsHandler struct {
	secretStore secrets.Store
	namespace   string
}

func (h *specsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	config, err := rest.InClusterConfig()
	if nil != err {
		utilapi.WriteError(
			r.Context(),
			w,
			errors.NewServerInClusterConfigError().WithCause(err))
		return
	}
	client, err := kubernetes.NewForConfig(config)
	if nil != err {
		utilapi.WriteError(
			r.Context(),
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
			r.Context(),
			w,
			errors.NewServerGetConfigMapError(key, h.namespace).WithCause(err))
		return
	}
	var spec interface{}
	if err := json.Unmarshal([]byte(configMap.Data["spec.json"]), &spec); nil != err {
		utilapi.WriteError(
			r.Context(),
			w,
			errors.NewServerConfigMapJSONError(key, h.namespace).WithCause(err))
		return
	}
	spec = h.expandSecrets(r.Context(), spec)
	utilapi.WriteObjectOK(r.Context(), w, spec)
}

func (h *specsHandler) expandSecrets(ctx context.Context, spec interface{}) interface{} {
	switch v := spec.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for _, elm := range v {
			result = append(result, h.expandSecrets(ctx, elm))
		}
		return result
	case map[string]interface{}:
		secretName := extractSecretName(v)
		if nil != secretName {
			sec, err := h.secretStore.Get(ctx, *secretName)
			if err != nil || nil == sec {
				log.Printf("failed to get secret=%v: %v", secretName, err)
				return ""
			}
			return sec.Value
		}
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = h.expandSecrets(ctx, val)
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
