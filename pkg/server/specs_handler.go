package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/jsonpath"
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
	spec = h.expandEmbedType(r.Context(), spec, h.createGetOutputs(client))
	utilapi.WriteObjectOK(r.Context(), w, spec)
}

type LazyGetOutputs func() (map[string]interface{}, error)

func (h *specsHandler) createGetOutputs(client *kubernetes.Clientset) LazyGetOutputs {
	outputs := make(map[string]interface{})
	var err error
	var once sync.Once
	return func() (map[string]interface{}, error) {
		once.Do(func() {
			configMap, kerr := client.CoreV1().ConfigMaps(h.namespace).Get("outputs", metav1.GetOptions{})
			if nil != kerr {
				if ! kerrors.IsNotFound(kerr) {
					err = kerr
				}
				return
			}
			for k, v := range configMap.Data {
				var decoded interface{}
				kerr = json.Unmarshal([]byte(v), &decoded)
				if nil != kerr {
					log.Printf("failed to decode '%s' key of %s/outputs ConfigMap as json: %+v",
						k, h.namespace)
					continue
				}
				outputs[k] = decoded
			}
		})
		return outputs, err
	}
}

func (h *specsHandler) expandEmbedType(ctx context.Context, spec interface{}, getOutputs LazyGetOutputs) interface{} {
	switch v := spec.(type) {
	case []interface{}:
		result := make([]interface{}, len(v))
		for index, elm := range v {
			result[index] = h.expandEmbedType(ctx, elm, getOutputs)
		}
		return result
	case map[string]interface{}:
		secretName := extractEmbedType(v, "Secret")
		if nil != secretName {
			sec, err := h.secretStore.Get(ctx, *secretName)
			if err != nil || nil == sec {
				log.Printf("failed to get secret=%s: %v", *secretName, err)
				return ""
			}
			return sec.Value
		}
		outputPtrExpr := extractEmbedType(v, "Output")
		if nil != outputPtrExpr {
			jpath := jsonpath.New("expression")
			err := jpath.Parse("{"+ *outputPtrExpr+"}")
			if err != nil {
				return fmt.Sprintf("Invalid JSONPath(%s): %s", *outputPtrExpr, err.Error())
			}
			outputs, err := getOutputs()
			if err != nil {
				log.Printf("failed to get outputs: %v", err)
				return ""
			}
			val, err := jpath.FindResults(outputs)
			if err != nil {
				return fmt.Sprintf("Evaluation of JSONPath(%s) failed: %s", *outputPtrExpr, err.Error())
			}
			for _, v := range val {
				for _, vv := range v {
					return vv.Interface()
				}
			}
			return nil
		}
		result := make(map[string]interface{})
		for key, val := range v {
			result[key] = h.expandEmbedType(ctx, val, getOutputs)
		}
		return result
	default:
		return v
	}

}

func extractEmbedType(obj map[string]interface{}, desiredType string) *string {
	if len(obj) != 2 {
		return nil
	}
	if ty, ok := obj["$type"].(string); !ok || desiredType != ty {
		return nil
	}
	name, ok := obj["name"].(string)
	if !ok || "" == name {
		return nil
	}
	return &name
}
