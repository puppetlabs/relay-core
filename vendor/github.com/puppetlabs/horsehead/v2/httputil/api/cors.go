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

type corsHandler struct {
	allowedHeaders       map[corsMatchable]struct{}
	allowedMethods       map[corsMatchable]struct{}
	allowedMethodsHeader string
}

func (ch *corsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
}

type CORSBuilder struct {
	allowedHeaders        map[string]struct{}
	allowedHeaderPrefixes map[string]struct{}
	allowedMethods        map[string]struct{}
}

func (cb *CORSBuilder) AllowHeaderPrefix(prefix string) *CORSBuilder {
	cb.allowedHeaderPrefixes[http.CanonicalHeaderKey(prefix)] = corsValue

	return cb
}

func (cb *CORSBuilder) AllowHeaders(headers ...string) *CORSBuilder {
	for _, header := range headers {
		cb.allowedHeaders[http.CanonicalHeaderKey(header)] = corsValue
	}

	return cb
}

func (cb *CORSBuilder) AllowMethods(methods ...string) *CORSBuilder {
	for _, method := range methods {
		cb.allowedMethods[strings.ToUpper(method)] = corsValue
	}

	return cb
}

func (cb *CORSBuilder) Build() http.Handler {
	ch := &corsHandler{
		allowedHeaders: make(map[corsMatchable]struct{}),
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

	return ch
}

func NewCORSBuilder() *CORSBuilder {
	return &CORSBuilder{
		allowedHeaders:        make(map[string]struct{}),
		allowedHeaderPrefixes: make(map[string]struct{}),
		allowedMethods:        make(map[string]struct{}),
	}
}
