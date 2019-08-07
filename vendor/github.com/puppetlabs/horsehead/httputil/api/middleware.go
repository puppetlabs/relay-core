package api

import (
	"net/http"
	"time"

	"github.com/puppetlabs/horsehead/logging"
	"github.com/puppetlabs/horsehead/request"
)

func RequestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ri := request.New()

		ctx := r.Context()
		ctx = request.NewContext(ctx, ri)
		ctx = logging.NewContext(ctx, "request", ri.Identifier)

		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

type durationMs time.Duration

func (d durationMs) Milliseconds() float64 {
	sec := time.Duration(d) / time.Millisecond
	nsec := time.Duration(d) % time.Millisecond
	return float64(sec) + float64(nsec)/1e6
}

func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trw := NewTrackingResponseWriter(w)

		start := time.Now()
		next.ServeHTTP(trw, r)
		duration := time.Now().Sub(start)

		if code, ok := trw.StatusCode(); !ok {
			// Probably hijacked, ignore.
		} else {
			attrs := logging.Ctx{
				"method":  r.Method,
				"path":    r.RequestURI,
				"status":  code,
				"time_ms": durationMs(duration).Milliseconds(),
			}

			if code >= 500 {
				log(r.Context()).Warn("handled HTTP request with server error", attrs)
			} else {
				log(r.Context()).Debug("handled HTTP request", attrs)
			}
		}
	})
}

func AccessControlMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("origin")

		switch origin {
		case "null":
			w.Header().Set("access-control-allow-origin", "*")
		case "":
		default:
			w.Header().Set("access-control-allow-origin", origin)
			w.Header().Set("access-control-allow-credentials", "true")
			w.Header().Add("vary", "Origin")
		}

		next.ServeHTTP(w, r)
	})
}
