package authenticate

import (
	"context"
	"net/http"
	"strings"
)

type HTTPAuthorizationHeaderIntermediary struct {
	r *http.Request
}

var _ Intermediary = &HTTPAuthorizationHeaderIntermediary{}

func (hi *HTTPAuthorizationHeaderIntermediary) Next(ctx context.Context, state *Authentication) (Raw, error) {
	hdr := hi.r.Header.Get("authorization")
	if !strings.HasPrefix(hdr, "Bearer ") {
		return nil, ErrNotFound
	}

	return Raw(strings.TrimPrefix(hdr, "Bearer ")), nil
}

func NewHTTPAuthorizationHeaderIntermediary(r *http.Request) *HTTPAuthorizationHeaderIntermediary {
	return &HTTPAuthorizationHeaderIntermediary{
		r: r,
	}
}
