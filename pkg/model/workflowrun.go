package model

import (
	"context"
	"net/url"

	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
)

type WorkflowRun struct {
	// Name is the name of the workflow that ran
	Name string `json:"name"`
	// RunNumber is the run number for the workflow
	RunNumber int32 `json:"run_number"`
	// URL is the server URL the run was requested on
	URL *url.URL `json:"url"`
}

type WorkflowRunManager interface {
	Run(ctx context.Context, name string, parameters map[string]openapi.WorkflowRunParameter) (*WorkflowRun, error)
}
