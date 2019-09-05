package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type ctxKey int

const (
	opsKey ctxKey = iota
)

type MiddlewareFunc func(http.Handler) http.Handler

func Managers(r *http.Request) op.ManagerFactory {
	ops, ok := r.Context().Value(opsKey).(op.ManagerFactory)
	if !ok {
		panic("no managers in request")
	}

	return newRequestScopedManagerFactory(r, ops)
}

func ManagerFactoryMiddleware(ops op.ManagerFactory) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), opsKey, ops)))
		})
	}
}

type requestScopedManagerFactory struct {
	delegate op.ManagerFactory
	mm       op.MetadataManager
}

func (m requestScopedManagerFactory) SecretsManager() op.SecretsManager {
	return m.delegate.SecretsManager()
}

func (m requestScopedManagerFactory) OutputsManager() op.OutputsManager {
	return m.delegate.OutputsManager()
}

func (m requestScopedManagerFactory) MetadataManager() op.MetadataManager {
	return m.mm
}

func (m requestScopedManagerFactory) KubernetesManager() op.KubernetesManager {
	return m.delegate.KubernetesManager()
}

func newRequestScopedManagerFactory(r *http.Request, ops op.ManagerFactory) *requestScopedManagerFactory {
	clientIP := r.Header.Get("X-Forwarded-For")
	parts := strings.Split(clientIP, ",")
	// use the head (first client ip observed)
	clientIP = parts[0]

	// if there's no forwarded header, then we just try to use the IP set by Go in the request.
	if clientIP == "" {
		parts := strings.Split(r.RemoteAddr, ":")
		clientIP = parts[0]
	}

	mm := ops.MetadataManager()
	if mm == nil {
		mm = task.NewPodMetadataManager(ops.KubernetesManager().Client(), task.PodMetadataManagerOptions{
			PodIP: clientIP,
		})
	}

	return &requestScopedManagerFactory{
		delegate: ops,
		mm:       mm,
	}
}
