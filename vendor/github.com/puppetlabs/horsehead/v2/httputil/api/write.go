package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/puppetlabs/errawr-go/v2/pkg/encoding"
	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/horsehead/v2/httputil/errors"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
)

type TrackingResponseWriter interface {
	http.ResponseWriter

	StatusCode() (int, bool)
	Committed() bool
}

type responseWriter struct {
	http.ResponseWriter

	statusCode int
}

func (rw *responseWriter) Write(data []byte) (n int, err error) {
	if rw.statusCode == 0 {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.ResponseWriter.Write(data)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.statusCode == 0 {
		rw.statusCode = statusCode
	}

	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) StatusCode() (int, bool) {
	return rw.statusCode, rw.statusCode != 0
}

func (rw *responseWriter) Committed() bool {
	_, ok := rw.StatusCode()
	return ok
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

func NewTrackingResponseWriter(delegate http.ResponseWriter) TrackingResponseWriter {
	tw := &responseWriter{ResponseWriter: delegate}

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

func WriteObjectWithStatus(ctx context.Context, w http.ResponseWriter, status int, object interface{}) {
	b, err := json.Marshal(object)
	if err != nil {
		WriteError(ctx, w, errors.NewAPIResourceSerializationError().WithCause(err).Bug())
		return
	}

	w.Header().Set("content-type", "application/json")

	// If this is a tagged object or envelope, write an ETag header for it.
	if status >= 200 && status < 400 {
		if cacheable, ok := object.(Cacheable); ok {
			if tag, ok := cacheable.CacheKey(); ok {
				w.Header().Set("etag", ETag{Value: tag}.String())
			}
		}
	}

	w.WriteHeader(status)

	if _, err := w.Write(b); err != nil {
		log(ctx).Error("Writing HTTP response failed.", "error", err)

		// Force this request to be abandoned.
		panic(err)
	}
}

func WriteObjectOK(ctx context.Context, w http.ResponseWriter, object interface{}) {
	WriteObjectWithStatus(ctx, w, http.StatusOK, object)
}

func WriteObjectCreated(ctx context.Context, w http.ResponseWriter, object interface{}) {
	WriteObjectWithStatus(ctx, w, http.StatusCreated, object)
}

type ErrorEnvelope struct {
	Error *encoding.ErrorDisplayEnvelope `json:"error"`
}

func NewErrorEnvelope(err errors.Error) *ErrorEnvelope {
	return &ErrorEnvelope{
		Error: encoding.ForDisplay(err),
	}
}

func NewErrorEnvelopeWithSensitivity(err errors.Error, sensitivity errawr.ErrorSensitivity) *ErrorEnvelope {
	return &ErrorEnvelope{
		Error: encoding.ForDisplayWithSensitivity(err, sensitivity),
	}
}

func WriteError(ctx context.Context, w http.ResponseWriter, err errors.Error) {
	status := http.StatusInternalServerError
	if hm, ok := err.Metadata().HTTP(); ok {
		status = hm.Status()

		for key, values := range hm.Headers() {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	if status == http.StatusInternalServerError || err.IsBug() {
		log(ctx).Error("internal error", "error", err)

		if a, ok := trackers.CapturerFromContext(ctx); ok {
			// OK for this to be async. If this fails, we'll still get it in the
			// logs.
			a.Capture(err).Report(ctx)
		}
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)

	var env *ErrorEnvelope
	if sensitivity, ok := ErrorSensitivityFromContext(ctx); ok {
		env = NewErrorEnvelopeWithSensitivity(err, sensitivity)
	} else {
		env = NewErrorEnvelope(err)
	}

	if err := json.NewEncoder(w).Encode(env); err != nil {
		log(ctx).Error("Writing HTTP response failed.", "error", err)

		// Force this request to be abandoned.
		panic(err)
	}
}

func WriteRedirect(w http.ResponseWriter, u *url.URL) {
	w.Header().Set("location", u.String())
	w.WriteHeader(http.StatusTemporaryRedirect)
}
