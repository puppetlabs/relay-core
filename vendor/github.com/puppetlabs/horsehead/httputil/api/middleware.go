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

type TrackingResponseWriter interface {
	http.ResponseWriter
	StatusCode() int
}

type trackingResponseWriter struct {
	http.ResponseWriter

	statusCode int
}

func (trw *trackingResponseWriter) Write(data []byte) (n int, err error) {
	if trw.statusCode == 0 {
		trw.WriteHeader(http.StatusOK)
	}

	return trw.ResponseWriter.Write(data)
}

func (trw *trackingResponseWriter) WriteHeader(statusCode int) {
	if trw.statusCode == 0 {
		trw.statusCode = statusCode
	}

	trw.ResponseWriter.WriteHeader(statusCode)
}

func (trw *trackingResponseWriter) StatusCode() int {
	return trw.statusCode
}

type responseWriterHijacker struct {
	TrackingResponseWriter
	http.Hijacker
}

type responseWriterFlusher struct {
	TrackingResponseWriter
	http.Flusher
}

type responseWriterHijackerFlusher struct {
	TrackingResponseWriter
	http.Hijacker
	http.Flusher
}

func newTrackingResponseWriter(delegate http.ResponseWriter) TrackingResponseWriter {
	tw := &trackingResponseWriter{ResponseWriter: delegate}

	if hijacker, ok := delegate.(http.Hijacker); ok {
		if flusher, ok := delegate.(http.Flusher); ok {
			return &responseWriterHijackerFlusher{
				TrackingResponseWriter: tw,
				Hijacker:               hijacker,
				Flusher:                flusher,
			}
		} else {
			return &responseWriterHijacker{
				TrackingResponseWriter: tw,
				Hijacker:               hijacker,
			}
		}
	} else if flusher, ok := delegate.(http.Flusher); ok {
		return &responseWriterFlusher{
			TrackingResponseWriter: tw,
			Flusher:                flusher,
		}
	}

	return tw
}

func LogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		trw := newTrackingResponseWriter(w)

		start := time.Now()
		next.ServeHTTP(trw, r)
		duration := time.Now().Sub(start)

		attrs := logging.Ctx{
			"method":  r.Method,
			"path":    r.RequestURI,
			"status":  trw.StatusCode(),
			"time_ms": durationMs(duration).Milliseconds(),
		}

		if trw.StatusCode() == 0 {
			// Probably hijacked, ignore.
			return
		} else if trw.StatusCode() >= 500 {
			log(r.Context()).Warn("handled HTTP request with server error", attrs)
		} else {
			log(r.Context()).Debug("handled HTTP request", attrs)
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
