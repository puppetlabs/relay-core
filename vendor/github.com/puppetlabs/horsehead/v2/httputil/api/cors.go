package api

import (
	"net/http"
	"strings"
)

var (
	corsValue = struct{}{}

	corsDefaultAllowedMethods       = []string{"OPTIONS", "GET", "HEAD", "POST"}
	corsDefaultAllowedMethodsHeader = strings.Join(corsDefaultAllowedMethods, ", ")
	corsDefaultAllowedMethodsM      = corsMatchableStrings(corsDefaultAllowedMethods)

	corsDefaultAllowedHeaders  = []string{"Accept", "Accept-Language", "Content-Language", "DPR", "Downlink", "Save-Data", "Save-Data", "Viewport-Width", "Width"}
	corsDefaultAllowedHeadersM = corsMatchableStrings(corsDefaultAllowedHeaders)
)

type corsMatchable interface {
	Match(candidate string) bool
}

type corsMatchableString string

func (cms corsMatchableString) Match(candidate string) bool {
	return candidate == string(cms)
}

func corsMatchableStrings(is []string) map[corsMatchable]struct{} {
	m := make(map[corsMatchable]struct{})

	for _, i := range is {
		m[corsMatchableString(i)] = corsValue
	}

	return m
}

type corsMatchablePrefix string

func (cmp corsMatchablePrefix) Match(candidate string) bool {
	return strings.HasPrefix(candidate, string(cmp))
}

func corsMatch(ms map[corsMatchable]struct{}, s string) bool {
	for m := range ms {
		if m.Match(s) {
			return true
		}
	}

	return false
}

type corsMiddleware struct {
	allowedOrigins       map[corsMatchable]struct{}
	defaultAllowedOrigin string
	next                 http.Handler
}

func (cm *corsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(cm.allowedOrigins) > 0 {
		origin := r.Header.Get("origin")

		if corsMatch(cm.allowedOrigins, origin) {
			w.Header().Set("access-control-allow-origin", origin)
		} else {
			w.Header().Set("access-control-allow-origin", cm.defaultAllowedOrigin)
		}

		w.Header().Set("vary", "Origin")
	}

	cm.next.ServeHTTP(w, r)
}

type corsPreflightHandler struct {
	allowedHeaders       map[corsMatchable]struct{}
	allowedMethods       map[corsMatchable]struct{}
	allowedMethodsHeader string
	allowedOrigins       map[corsMatchable]struct{}
	defaultAllowedOrigin string
}

func (ch *corsPreflightHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestedMethod := strings.ToUpper(r.Header.Get("access-control-request-method"))
	if requestedMethod == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	} else if !corsMatch(ch.allowedMethods, requestedMethod) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	requestedHeadersL := r.Header[http.CanonicalHeaderKey("access-control-request-headers")]
	var allowedHeaders []string
	for _, requestedHeadersS := range requestedHeadersL {
		requestedHeaders := strings.Split(requestedHeadersS, ",")
		for _, requestedHeader := range requestedHeaders {
			canonicalHeader := http.CanonicalHeaderKey(strings.TrimSpace(requestedHeader))
			if canonicalHeader == "" || corsMatch(corsDefaultAllowedHeadersM, canonicalHeader) {
				continue
			}

			if !corsMatch(ch.allowedHeaders, canonicalHeader) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			allowedHeaders = append(allowedHeaders, canonicalHeader)
		}
	}

	w.Header().Set("access-control-allow-methods", ch.allowedMethodsHeader)

	if len(allowedHeaders) > 0 {
		w.Header().Set("access-control-allow-headers", strings.Join(allowedHeaders, ", "))
	}

	if len(ch.allowedOrigins) > 0 {
		origin := r.Header.Get("origin")

		if corsMatch(ch.allowedOrigins, origin) {
			w.Header().Set("access-control-allow-origin", origin)
		} else {
			w.Header().Set("access-control-allow-origin", ch.defaultAllowedOrigin)
		}

		w.Header().Set("vary", "Origin")
	}
}

// CORSBuilder builds an http.Handler that can be used as a middleware to set
// CORS access control headers for relaxing same-origin browser policies.
type CORSBuilder struct {
	allowedHeaders        map[string]struct{}
	allowedHeaderPrefixes map[string]struct{}
	allowedMethods        map[string]struct{}
	allowedOrigins        map[string]struct{}
	defaultAllowedOrigin  string
}

// AllowHeaderPrefix takes a header prefix and allows all headers that
// match the given prefix for requests.
//
// example AllowHeaderPrefix("example-") will allow a request with header
// Example-XYZ.
func (cb *CORSBuilder) AllowHeaderPrefix(prefix string) *CORSBuilder {
	cb.allowedHeaderPrefixes[http.CanonicalHeaderKey(prefix)] = corsValue

	return cb
}

// AllowHeaders takes a variadic of header strings to allow. It is similar
// to AllowHeaderPrefix, but it matches against the entire string instead
// of matching against a partial prefix.
func (cb *CORSBuilder) AllowHeaders(headers ...string) *CORSBuilder {
	for _, header := range headers {
		cb.allowedHeaders[http.CanonicalHeaderKey(header)] = corsValue
	}

	return cb
}

// AllowMethods takes a variadic of http methods to allow.
func (cb *CORSBuilder) AllowMethods(methods ...string) *CORSBuilder {
	for _, method := range methods {
		cb.allowedMethods[strings.ToUpper(method)] = corsValue
	}

	return cb
}

// AllowOrigins takes a variadic of http origins to allow. The match is against
// the entire origin string and no patterns are allowed. The first one in the list
// is the default origin to return in the event the origin in the request isn't.
// There is no attempt to error on an origin that isn't in the list because this is
// the client's job. We simple return an origin that _is_ allowed and let the client
// block the request from happening.
func (cb *CORSBuilder) AllowOrigins(origins ...string) *CORSBuilder {
	for _, origin := range origins {
		cb.allowedOrigins[origin] = corsValue

		if cb.defaultAllowedOrigin == "" {
			cb.defaultAllowedOrigin = origin
		}
	}

	return cb
}

// PreflightHandler returns an http.Handler that can set Access-Control-Allow-* headers
// for preflight-requests (OPTIONS).
func (cb *CORSBuilder) PreflightHandler() http.Handler {
	return cb.Build()
}

// Middleware wraps an http.Handler to set ACAO headers on responses.
func (cb *CORSBuilder) Middleware(next http.Handler) http.Handler {
	cm := &corsMiddleware{
		allowedOrigins:       make(map[corsMatchable]struct{}),
		defaultAllowedOrigin: cb.defaultAllowedOrigin,
	}

	for origin := range cb.allowedOrigins {
		cm.allowedOrigins[corsMatchableString(origin)] = corsValue
	}

	cm.defaultAllowedOrigin = cb.defaultAllowedOrigin

	cm.next = next

	return cm
}

// Build returns an http.Handler that can set Access-Control-Allow-* headers
// based on requests it receives.
//
// DEPRECATED use PreflightHandler.
func (cb *CORSBuilder) Build() http.Handler {
	ch := &corsPreflightHandler{
		allowedHeaders: make(map[corsMatchable]struct{}),
		allowedOrigins: make(map[corsMatchable]struct{}),
	}

	for allowedHeader := range cb.allowedHeaders {
		ch.allowedHeaders[corsMatchableString(allowedHeader)] = corsValue
	}

	for allowedHeaderPrefix := range cb.allowedHeaderPrefixes {
		ch.allowedHeaders[corsMatchablePrefix(allowedHeaderPrefix)] = corsValue
	}

	if len(cb.allowedMethods) == 0 {
		ch.allowedMethods = corsDefaultAllowedMethodsM
		ch.allowedMethodsHeader = corsDefaultAllowedMethodsHeader
	} else {
		var allowedMethods []string
		if _, found := cb.allowedMethods["OPTIONS"]; !found {
			allowedMethods = append(allowedMethods, "OPTIONS")
		}

		for allowedMethod := range cb.allowedMethods {
			allowedMethods = append(allowedMethods, allowedMethod)
		}

		ch.allowedMethods = corsMatchableStrings(allowedMethods)
		ch.allowedMethodsHeader = strings.Join(allowedMethods, ", ")
	}

	for origin := range cb.allowedOrigins {
		ch.allowedOrigins[corsMatchableString(origin)] = corsValue
	}

	ch.defaultAllowedOrigin = cb.defaultAllowedOrigin

	return ch
}

// NewCORSBuilder returns a new CORSBuilder.
func NewCORSBuilder() *CORSBuilder {
	return &CORSBuilder{
		allowedHeaders:        make(map[string]struct{}),
		allowedHeaderPrefixes: make(map[string]struct{}),
		allowedMethods:        make(map[string]struct{}),
		allowedOrigins:        make(map[string]struct{}),
	}
}
