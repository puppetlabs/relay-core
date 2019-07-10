package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/puppetlabs/errawr-go/v2/pkg/encoding"
	"github.com/puppetlabs/horsehead/httputil/errors"
	"github.com/puppetlabs/horsehead/instrumentation/alerts/trackers"
)

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
	if err := json.NewEncoder(w).Encode(NewErrorEnvelope(err)); err != nil {
		log(ctx).Error("Writing HTTP response failed.", "error", err)

		// Force this request to be abandoned.
		panic(err)
	}
}

func WriteRedirect(w http.ResponseWriter, u *url.URL) {
	w.Header().Set("location", u.String())
	w.WriteHeader(http.StatusTemporaryRedirect)
}
