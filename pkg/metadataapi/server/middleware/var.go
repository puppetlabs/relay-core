package middleware

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

func Var(r *http.Request, name string) (string, bool) {
	encoded, found := mux.Vars(r)[name]
	if !found {
		return "", false
	}

	value, err := url.QueryUnescape(encoded)
	if err != nil {
		return "", false
	}

	return value, true
}
