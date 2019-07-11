package api

import (
	"context"
	"fmt"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/puppetlabs/horsehead/httputil/errors"
)

type Cacheable interface {
	CacheKey() (string, bool)
}

type ETag struct {
	Weak  bool
	Value string
}

func (et ETag) String() string {
	encoded := strconv.QuoteToASCII(et.Value)

	if et.Weak {
		return fmt.Sprintf("W/%s", encoded)
	}

	return encoded
}

type ConditionalResolver interface {
	Accept(ctx context.Context, w http.ResponseWriter, c Cacheable) bool
}

type statusConditionalResolver struct {
	status int
}

func (scr *statusConditionalResolver) Accept(ctx context.Context, w http.ResponseWriter, c Cacheable) bool {
	w.WriteHeader(scr.status)
	return false
}

type errorConditionalResolver struct {
	err errors.Error
}

func (ecr *errorConditionalResolver) Accept(ctx context.Context, w http.ResponseWriter, c Cacheable) bool {
	WriteError(ctx, w, ecr.err)
	return false
}

type noOpConditionalResolver struct{}

func (nocr *noOpConditionalResolver) Accept(ctx context.Context, w http.ResponseWriter, c Cacheable) bool {
	return true
}

type ifMatchConditionalResolver struct {
	tags     []ETag
	delegate ConditionalResolver
}

func (imcr *ifMatchConditionalResolver) Accept(ctx context.Context, w http.ResponseWriter, c Cacheable) bool {
	if len(imcr.tags) == 0 {
		if c == nil {
			// We requested `If-Match: *`, but this resource representation
			// doesn't exist.
			return imcr.delegate.Accept(ctx, w, c)
		}

		// Otherwise, we match any object.
		return true
	} else if c == nil {
		// We have requested at least one tag to match a nonexistent resource
		// representation, which will never succeed.
		return imcr.delegate.Accept(ctx, w, c)
	}

	if tag, ok := c.CacheKey(); ok {
		for _, candidate := range imcr.tags {
			if candidate.Value == tag {
				return true
			}
		}
	}

	return imcr.delegate.Accept(ctx, w, c)
}

type ifNoneMatchConditionalResolver struct {
	tags     []ETag
	delegate ConditionalResolver
}

func (inmcr *ifNoneMatchConditionalResolver) Accept(ctx context.Context, w http.ResponseWriter, c Cacheable) bool {
	if len(inmcr.tags) == 0 {
		// `If-None-Match: *`` specified. The only case where this should be
		// accepted is if the requested representation is nonexistent.

		if c == nil {
			return true
		}

		return inmcr.delegate.Accept(ctx, w, c)
	} else if c == nil {
		// In this case, we have a list of tags we want to match, but no entity
		// to match them exist. We allow this because, for whatever reason, our
		// caller has selected `nil` as the resource representation.
		return true
	}

	if tag, ok := c.CacheKey(); ok {
		for _, candidate := range inmcr.tags {
			if candidate.Value == tag {
				return inmcr.delegate.Accept(ctx, w, c)
			}
		}
	}

	return true
}

func NewConditionalResolver(r *http.Request) ConditionalResolver {
	if ims := r.Header["If-Match"]; len(ims) > 0 {
		var delegate ConditionalResolver
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			delegate = &errorConditionalResolver{errors.NewAPICachedResourceNotAvailableError()}
		default:
			delegate = &errorConditionalResolver{errors.NewAPIResourceModifiedError()}
		}

		tags, ok := scanETags(ims)
		if !ok {
			return delegate
		}

		return &ifMatchConditionalResolver{
			tags:     tags,
			delegate: delegate,
		}
	} else if inms := r.Header["If-None-Match"]; len(inms) > 0 {
		var delegate ConditionalResolver
		switch r.Method {
		case http.MethodGet, http.MethodHead:
			delegate = &statusConditionalResolver{http.StatusNotModified}
		default:
			delegate = &errorConditionalResolver{errors.NewAPIResourceModifiedError()}
		}

		tags, ok := scanETags(inms)
		if !ok {
			return delegate
		}

		return &ifNoneMatchConditionalResolver{
			tags:     tags,
			delegate: delegate,
		}
	}

	return &noOpConditionalResolver{}
}

func scanETags(hs []string) ([]ETag, bool) {
	var tags []ETag

	for _, h := range hs {
		for {
			h = textproto.TrimString(h)
			if len(h) == 0 {
				break
			} else if h[0] == ',' {
				h = h[1:]
				continue
			} else if h[0] == '*' {
				return []ETag{}, true
			}

			tag, rest := scanETag(h)
			if tag.Value == "" {
				// This is an invalid header.
				return nil, false
			}

			tags = append(tags, tag)
			h = rest
		}
	}

	return tags, true
}

func scanETag(h string) (tag ETag, rest string) {
	start := 0
	if strings.HasPrefix(h, "W/") {
		tag.Weak = true
		start += 2
	}

	if len(h[start:]) < 2 || h[start] != '"' {
		return
	}

	for i := start + 1; i < len(h); i++ {
		if h[i] == '"' {
			tag.Value = h[start+1 : i]
			rest = h[i+1:]
			return
		}

		if (h[i] < 0x23 && h[i] != '!') || (h[i] > 0x7e && h[i] < 0x80) {
			return
		}
	}

	return
}
