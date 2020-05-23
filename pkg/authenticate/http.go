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
	if username, password, ok := hi.r.BasicAuth(); ok {
		if username != "" {
			return nil, &NotFoundError{Reason: "http: username not empty"}
		}

		return Raw(password), nil
	} else if token, ok := parseBearerAuth(hi.r.Header.Get("authorization")); ok {
		return Raw(token), nil
	}

	return nil, &NotFoundError{Reason: "http: neither Basic nor Bearer authentication present"}
}

func NewHTTPAuthorizationHeaderIntermediary(r *http.Request) *HTTPAuthorizationHeaderIntermediary {
	return &HTTPAuthorizationHeaderIntermediary{
		r: r,
	}
}

func parseBearerAuth(auth string) (token string, ok bool) {
	const prefix = "Bearer "
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}

	return auth[len(prefix):], true
}
