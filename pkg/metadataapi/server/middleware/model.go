package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/relay-core/pkg/manager/builder"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type modelContextKey int

const (
	managersModelContextKey modelContextKey = iota
)

func Managers(r *http.Request) model.MetadataManagers {
	mgrs, ok := r.Context().Value(managersModelContextKey).(model.MetadataManagers)
	if !ok {
		return builder.NewMetadataBuilder().Build()
	}

	return mgrs
}

func WithManagers(m model.MetadataManagers) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), managersModelContextKey, m))
			next.ServeHTTP(w, r)
		})
	}
}
