package api

import (
	"net/http"
	"time"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

func (s *Server) PutTimer(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	ctx := r.Context()

	managers := middleware.Managers(r)
	tm := managers.Timers()

	name, _ := middleware.Var(r, "name")

	if _, err := tm.Set(ctx, name, now); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}
