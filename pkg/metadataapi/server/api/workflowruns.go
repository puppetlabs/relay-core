package api

import (
	"encoding/json"
	"net/http"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
)

type PostWorkflowRunRequestEnvelope struct {
	Parameters map[string]openapi.WorkflowRunParameter `json:"parameters"`
}

type WorkflowRunEnvelope struct {
	Name      string `json:"name"`
	RunNumber int    `json:"run_number"`
	AppURL    string `json:"app_url,omitempty"`
}

type PostWorkflowRunResponseEnvelope struct {
	WorkflowRun *WorkflowRunEnvelope `json:"workflow_run"`
}

func (s *Server) PostWorkflowRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	managers := middleware.Managers(r)
	wrm := managers.WorkflowRuns()

	var env PostWorkflowRunRequestEnvelope
	if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
		utilapi.WriteError(ctx, w, errors.NewAPIMalformedRequestError().WithCause(err))

		return
	}

	name, _ := middleware.Var(r, "name")

	wr, err := wrm.Run(ctx, name, env.Parameters)
	if err != nil {
		utilapi.WriteError(ctx, w, ModelWriteError(err))

		return
	}

	resp := PostWorkflowRunResponseEnvelope{
		WorkflowRun: &WorkflowRunEnvelope{
			Name:      wr.Name,
			RunNumber: wr.RunNumber,
		},
	}

	if wr.AppURL != nil {
		resp.WorkflowRun.AppURL = wr.AppURL.String()
	}

	utilapi.WriteObjectCreated(ctx, w, resp)
}
