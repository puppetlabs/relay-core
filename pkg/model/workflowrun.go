package model

import (
	"context"

	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
)

type WorkflowRun struct {
	// Name is the name of the workflow that ran
	Name string
	// RunNum is the run number for the workflow
	RunNum int32
	// URL is the server URL the run was requested on
	URL string
}

type WorkflowRunManager interface {
	Run(ctx context.Context, name string, parameters map[string]openapi.WorkflowRunParameter) (*WorkflowRun, error)
}
