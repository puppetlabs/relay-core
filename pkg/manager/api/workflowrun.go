package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
	"github.com/puppetlabs/relay-core/pkg/model"
)

var _ model.WorkflowRunManager = &WorkflowRunManager{}

type WorkflowRunManager struct {
	client *openapi.APIClient
	token  string
}

func (w *WorkflowRunManager) Run(ctx context.Context, name string, parameters map[string]openapi.WorkflowRunParameter) (*model.WorkflowRun, error) {
	ctx = context.WithValue(ctx, openapi.ContextAccessToken, w.token)

	wrc := w.client.WorkflowRunsApi
	req := wrc.RunWorkflow(ctx, name).CreateWorkflowRun(openapi.CreateWorkflowRun{
		Parameters: &parameters,
	})

	ent, resp, err := req.Execute()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := &UnexpectedResponseError{
			StatusCode: resp.StatusCode,
		}

		var env utilapi.ErrorEnvelope
		if derr := json.NewDecoder(resp.Body).Decode(&env); derr == nil && env.Error != nil {
			err.Cause = env.Error.AsError()
		}

		return nil, err
	}

	return &model.WorkflowRun{
		Name:   name,
		RunNum: ent.Run.RunNumber,
	}, nil
}

func NewWorkflowRunManager(us, token string) (*WorkflowRunManager, error) {
	u, err := url.Parse(us)
	if err != nil {
		return nil, err
	}

	cfg := openapi.NewConfiguration()
	cfg.Host = u.Host
	cfg.Scheme = u.Scheme

	client := openapi.NewAPIClient(cfg)

	return &WorkflowRunManager{
		client: client,
		token:  token,
	}, nil
}
