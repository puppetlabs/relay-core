package api

import (
	"net/http"
	"strings"
)

var (
	cspQuotableSources = map[string]struct{}{
		"none":           {},
		"self":           {},
		"unsafe-eval":    {},
		"unsafe-hashes":  {},
		"unsafe-inline":  {},
		"strict-dynamic": {},
		"report-sample":  {},
	}
)

func cspQuoteSources(sources []string) []string {
	var newSources = make([]string, len(sources))

	for i := range sources {
		src := sources[i]
		if _, ok := cspQuotableSources[sources[i]]; ok {
			src = "'" + sources[i] + "'"
		}

		newSources[i] = src
	}

	return newSources
}

type cspDirectiveType int

// This is an incomplete list of directives for the
// Content-Security-Policy header. More directives can
// be added here for further CSP support.
const (
	CSPBaseURI cspDirectiveType = iota
	CSPBlockAllMixedContent
	CSPDefaultSrc
	CSPFontSrc
	CSPFormSrc
	CSPFrameAncestors
	CSPFrameSrc
	CSPManifestSrc
	CSPMediaSrc
	CSPScriptSrc
	CSPStyleSrc
	CSPImgSrc
)

var cspMapping = map[cspDirectiveType]func(sources []string) string{
	CSPBaseURI: baseSourceDirectiveFactory("base-uri"),
	CSPBlockAllMixedContent: func(_ []string) string {
		return "block-all-mixed-content"
	},
	CSPDefaultSrc:     baseSourceDirectiveFactory("default-src"),
	CSPFontSrc:        baseSourceDirectiveFactory("font-src"),
	CSPFormSrc:        baseSourceDirectiveFactory("form-src"),
	CSPFrameAncestors: baseSourceDirectiveFactory("frame-ancestors"),
	CSPFrameSrc:       baseSourceDirectiveFactory("frame-src"),
	CSPManifestSrc:    baseSourceDirectiveFactory("manifest-src"),
	CSPMediaSrc:       baseSourceDirectiveFactory("media-src"),
	CSPScriptSrc:      baseSourceDirectiveFactory("script-src"),
	CSPStyleSrc:       baseSourceDirectiveFactory("style-src"),
	CSPImgSrc:         baseSourceDirectiveFactory("img-src"),
}

func baseSourceDirectiveFactory(directiveName string) func([]string) string {
	return func(sources []string) string {
		sources = cspQuoteSources(sources)

		s := []string{directiveName}
		s = append(s, sources...)

		return strings.Join(s, " ")
	}
}

// CSPBuilder uses values stored for src's and builds a valid
// Content-Security-Policy header.
type CSPBuilder struct {
	directives map[cspDirectiveType]string
	order      []cspDirectiveType
}

// SetDirective sets sources for directive types. Sources is a variadic and not every
// directive type uses them, so they are not always required.
func (cb *CSPBuilder) SetDirective(dt cspDirectiveType, sources ...string) *CSPBuilder {
	if cb.directives == nil {
		cb.directives = make(map[cspDirectiveType]string)
	}

	result := cspMapping[dt](sources)

	if _, ok := cb.directives[dt]; !ok {
		cb.order = append(cb.order, dt)
	}

	cb.directives[dt] = result

	return cb
}

// Middleware is a middleware wrapper that can be used to inject a
// Content-Security-Policy header into a server's response. It builds
// the header value and joins it together in the proper format then passes
// the request off to the `next` http.Handler.
func (cb *CSPBuilder) Middleware(next http.Handler) http.Handler {
	var policy string

	if len(cb.directives) > 0 {
		directives := []string{}

		for _, d := range cb.order {
			directives = append(directives, cb.directives[d])
		}

		policy = strings.Join(directives, "; ")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if policy != "" {
			w.Header().Set("content-security-policy", policy)
		}

		next.ServeHTTP(w, r)
	})
}
