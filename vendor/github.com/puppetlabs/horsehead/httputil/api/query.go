package api

import (
	"net/http"
	"net/url"
)

type SetQueryFunc func(out url.Values, in *http.Request)

func SetQuery(u *url.URL, r *http.Request, fns ...SetQueryFunc) {
	out := make(url.Values)
	for _, fn := range fns {
		fn(out, r)
	}

	u.RawQuery = out.Encode()
}
