package middleware

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	utilapi "github.com/puppetlabs/leg/httputil/api"
)

func WithErrorSensitivity(sensitivity errawr.ErrorSensitivity) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(utilapi.NewContextWithErrorSensitivity(r.Context(), sensitivity))
			next.ServeHTTP(w, r)
		})
	}
}
