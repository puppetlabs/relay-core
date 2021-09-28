package reject

import (
	"context"

	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
	"github.com/puppetlabs/relay-core/pkg/model"
)

type workflowRunManager struct{}

func (*workflowRunManager) Run(ctx context.Context, name string, parameters map[string]openapi.WorkflowRunParameter) (*model.WorkflowRun, error) {
	return nil, model.ErrRejected
}

var WorkflowRunManager model.WorkflowRunManager = &workflowRunManager{}
