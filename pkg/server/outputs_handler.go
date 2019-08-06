package server

import (
	"encoding/json"
	"net/http"
	"time"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

type outputsHandler struct {
	namespace string
}

func (h *outputsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if key != "" || r.Method != http.MethodPut {
		http.NotFound(w, r)
		return
	}

	var body = make(map[string]interface{})
	err = json.NewDecoder(r.Body).Decode(&body)
	if nil != err {
		utilapi.WriteError(
			r.Context(),
			w,
			errors.NewServerInvalidJSONBodyError().WithCause(err))
		return
	}

	for retry := uint(0); ; retry++ {
		var update = true
		configMap, err := client.CoreV1().ConfigMaps(h.namespace).Get("outputs", metav1.GetOptions{})
		if kerrors.IsNotFound(err) {
			update = false
			configMap = &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outputs",
					Namespace: h.namespace,
				},
			}
		} else if nil != err {
			utilapi.WriteError(
				r.Context(),
				w,
				errors.NewServerGetConfigMapError("outputs", h.namespace).WithCause(err))
			return
		}
		if nil == configMap.Data {
			configMap.Data = make(map[string]string)
		}
		for k, v := range body {
			strV, err := json.Marshal(v)
			if nil != err {
				utilapi.WriteError(
					r.Context(),
					w,
					errors.NewServerJSONMarshalError().WithCause(err))
				return
			}
			configMap.Data[k] = string(strV)
		}
		if update {
			configMap, err = client.CoreV1().ConfigMaps(h.namespace).Update(configMap)
			if nil == err {
				break
			} else if !kerrors.IsConflict(err) || retry > 7 {
				utilapi.WriteError(
					r.Context(),
					w,
					errors.NewServerUpdateConfigMapError(
						"outputs", h.namespace).WithCause(err))
				return
			}
			klog.Warningf("Conflict during ConfigMap.output update")
		} else {
			configMap, err = client.CoreV1().ConfigMaps(h.namespace).Create(configMap)
			if nil == err {
				break
			} else if !kerrors.IsAlreadyExists(err) || retry > 7 {
				utilapi.WriteError(
					r.Context(),
					w,
					errors.NewServerCreateConfigMapError(
						"outputs", h.namespace).WithCause(err))
				return
			}
			klog.Warningf("Already exists during ConfigMap.output create")
		}
		select {
		case <-r.Context().Done():
			klog.Warningf("Request cancelled: %+v", r.Context().Err())
			return
		case <-time.After((1 << retry) * 10 * time.Millisecond):
		}
	}
	utilapi.WriteObjectOK(r.Context(), w, struct {
		Ok bool `json:"ok"`
	}{true})
}
