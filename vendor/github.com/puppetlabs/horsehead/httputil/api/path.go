package api

import "net/url"

func NewPath(escaped string) *url.URL {
	unescaped, err := url.PathUnescape(escaped)
	if err != nil {
		return &url.URL{
			Path: escaped,
		}
	}

	return &url.URL{
		RawPath: escaped,
		Path:    unescaped,
	}
}
