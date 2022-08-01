package api

import (
	"encoding/json"
	"net/http"
	"time"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type ActionStatusProcessState struct {
	ExitCode  int       `json:"exit_code"`
	Timestamp time.Time `json:"timestamp"`
}

type ActionStatusWhenCondition struct {
	Timestamp           time.Time                 `json:"timestamp"`
	WhenConditionStatus model.WhenConditionStatus `json:"when_condition_status"`
}

type PutActionStatusRequestEnvelope struct {
	ProcessState  *ActionStatusProcessState  `json:"process_state"`
	WhenCondition *ActionStatusWhenCondition `json:"when_condition"`
}

func (s *Server) PutActionStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	asm := managers.ActionStatus()

	var env PutActionStatusRequestEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))
		return
	}

	if err := asm.Set(ctx, mapActionStatus(env)); err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func mapActionStatus(env PutActionStatusRequestEnvelope) *model.ActionStatus {
	as := &model.ActionStatus{}

	if env.ProcessState != nil {
		as.ProcessState = &model.ActionStatusProcessState{
			ExitCode:  env.ProcessState.ExitCode,
			Timestamp: env.ProcessState.Timestamp,
		}
	}

	if env.WhenCondition != nil {
		as.WhenCondition = &model.ActionStatusWhenCondition{
			Timestamp:           env.WhenCondition.Timestamp,
			WhenConditionStatus: env.WhenCondition.WhenConditionStatus,
		}
	}

	return as
}
