package app

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkflowDepsLoadResult struct {
	Upstream bool
	All      bool
}

// WorkflowDeps represents the dependencies of a Workflow.
type WorkflowDeps struct {
	Workflow   *obj.Workflow
	Tenant     *obj.Tenant
	TenantDeps *TenantDeps
}

func (wd *WorkflowDeps) Load(ctx context.Context, cl client.Client) (*WorkflowDepsLoadResult, error) {
	if ok, err := wd.Tenant.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		return &WorkflowDepsLoadResult{}, nil
	}

	wd.TenantDeps = NewTenantDeps(wd.Tenant)

	if ok, err := wd.TenantDeps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		return &WorkflowDepsLoadResult{}, nil
	}

	return &WorkflowDepsLoadResult{
		Upstream: true,
		All:      true,
	}, nil
}

func NewWorkflowDeps(w *obj.Workflow) *WorkflowDeps {
	key := w.Key

	wrd := &WorkflowDeps{
		Workflow: w,

		Tenant: obj.NewTenant(client.ObjectKey{
			Namespace: key.Namespace,
			Name:      w.Object.Spec.TenantRef.Name,
		}),
	}

	return wrd
}
