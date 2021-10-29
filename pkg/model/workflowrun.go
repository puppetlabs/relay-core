package model

import (
	"context"
	"net/url"

	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
)

type WorkflowRun struct {
	// Name is the name of the workflow that ran
	Name string
	// RunNumber is the run number for the workflow
	RunNumber int
	// AppURL is the frontend URL to the run
	AppURL *url.URL
}

type WorkflowRunManager interface {
	Run(ctx context.Context, name string, parameters map[string]openapi.WorkflowRunParameter) (*WorkflowRun, error)
}
