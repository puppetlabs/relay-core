package middleware

import (
	"context"
	"net"
	"net/http"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type ctxKey int

const (
	opsKey ctxKey = iota
	taskMetadataKey
)

type MiddlewareFunc func(http.Handler) http.Handler

func Managers(r *http.Request) op.ManagerFactory {
	ops, ok := r.Context().Value(opsKey).(op.ManagerFactory)
	if !ok {
		panic("no managers in request")
	}

	return ops
}

func TaskMetadata(r *http.Request) *task.Metadata {
	md, ok := r.Context().Value(taskMetadataKey).(*task.Metadata)
	if !ok {
		panic("no task metadata in request")
	}

	return md
}

func ManagerFactoryMiddleware(ops op.ManagerFactory) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), opsKey, ops)))
		})
	}
}

func TaskMetadataMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ops := Managers(r)

		ctx := r.Context()

		// as long as we are using the standard lib http server, it will fill in the RemoteAddr field as
		// host:port and we will be able to pull the clientIP out. If we end up using a 3rd party package
		// as the http server (rare), then the handler will raise an error if it uses a differently formatted
		// RemoteAddr.
		clientIP, _, goerr := net.SplitHostPort(r.RemoteAddr)
		if goerr != nil {
			utilapi.WriteError(ctx, w, errors.NewServerClientIPError().WithCause(goerr))

			return
		}

		md, err := ops.MetadataManager().GetByIP(ctx, clientIP)
		if err != nil {
			utilapi.WriteError(ctx, w, err)

			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), taskMetadataKey, md)))
	})
}
