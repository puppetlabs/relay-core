package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"

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

	newOps, err := newRequestScopedManagerFactory(r, ops)
	if err != nil {
		panic(fmt.Sprintf("malformed or missing remote address: %v", err))
	}

	return newOps
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

func newRequestScopedManagerFactory(r *http.Request, ops op.ManagerFactory) (*requestScopedManagerFactory, error) {
	// as long as we are using the standard lib http server, it will fill in the RemoteAddr field as
	// host:port and we will be able to pull the clientIP out. If we end up using a 3rd party package
	// as the http server (rare), then the caller above will panic if it uses a differently formatted
	// RemoteAddr.
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
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
	}, nil
}
