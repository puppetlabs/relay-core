package server

import (
	"net/http"
	"strings"

	utilapi "github.com/puppetlabs/horsehead/httputil/api"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets"
)

type secretsHandler struct {
	sec secrets.Store
}

func (h *secretsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenParts := strings.Split(authHeader, "Bearer ")

	if len(tokenParts) != 2 {
		w.WriteHeader(http.StatusUnauthorized)

		return
	}

	token := tokenParts[1]

	var wn, tn, key string

	// TODO clean this path validation logic up
	wn, r.URL.Path = shiftPath(r.URL.Path)
	if wn == "" || wn == "/" {
		http.NotFound(w, r)

		return
	}

	tn, r.URL.Path = shiftPath(r.URL.Path)
	if tn == "" || tn == "/" {
		http.NotFound(w, r)

		return
	}

	key, r.URL.Path = shiftPath(r.URL.Path)
	if key == "" || key == "/" {
		http.NotFound(w, r)

		return
	}

	sess, err := h.sec.GetScopedSession(wn, tn, token)
	if err != nil {
		utilapi.WriteError(r.Context(), w, err)

		return
	}

	sec, err := sess.Get(r.Context(), key)
	if err != nil {
		utilapi.WriteError(r.Context(), w, err)

		return
	}

	utilapi.WriteObjectOK(r.Context(), w, sec)
}
