package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func WithTrustedProxyHops(n int) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var hops []string
			for _, hdr := range r.Header[http.CanonicalHeaderKey("x-forwarded-for")] {
				hops = append(hops, strings.Split(hdr, ",")...)
			}

			for i := len(hops) - 1; i >= 0 && i >= len(hops)-n; i-- {
				hop := net.ParseIP(strings.TrimSpace(hops[i]))
				if hop == nil {
					break
				}

				r.RemoteAddr = hop.String()
			}

			next.ServeHTTP(w, r)
		})
	}
}
