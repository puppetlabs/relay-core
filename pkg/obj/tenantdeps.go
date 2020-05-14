package obj

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TenantDeps interface {
	Persister
	Loader

	configure(ctx context.Context)
}

type UnmanagedTenantDeps struct {
	Tenant    *Tenant
	Namespace *Namespace
}

var _ TenantDeps = &UnmanagedTenantDeps{}

func (utd *UnmanagedTenantDeps) Persist(ctx context.Context, cl client.Client) error {
	return nil
}

func (utd *UnmanagedTenantDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	return RequiredLoader{utd.Namespace}.Load(ctx, cl)
}

func (utd *UnmanagedTenantDeps) configure(ctx context.Context) {}

type ManagedTenantDeps struct {
	Tenant *Tenant

	Namespace     *Namespace
	NetworkPolicy *NetworkPolicy
}

var _ TenantDeps = &ManagedTenantDeps{}

func (mtd *ManagedTenantDeps) Persist(ctx context.Context, cl client.Client) error {
	ps := []Persister{
		mtd.Namespace,
		mtd.NetworkPolicy,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (mtd *ManagedTenantDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	return Loaders{
		mtd.Namespace,
		mtd.NetworkPolicy,
	}.Load(ctx, cl)
}

func (mtd *ManagedTenantDeps) configure(ctx context.Context) {
	os := []Ownable{
		mtd.Namespace,
		mtd.NetworkPolicy,
	}
	for _, o := range os {
		mtd.Tenant.Own(ctx, o)
	}

	mtd.Namespace.Label(ctx, model.RelayControllerTenantWorkloadLabel, "true")
	mtd.Namespace.LabelAnnotateFrom(ctx, mtd.Tenant.Object.Spec.NamespaceTemplate.Metadata)

	ConfigureNetworkPolicyForTenant(mtd.NetworkPolicy)
}

func NewTenantDeps(t *Tenant) TenantDeps {
	if !t.Managed() {
		return &UnmanagedTenantDeps{
			Tenant:    t,
			Namespace: NewNamespace(t.Key.Namespace),
		}
	}

	ns := t.Object.Spec.NamespaceTemplate.Metadata.GetName()

	return &ManagedTenantDeps{
		Tenant: t,

		Namespace:     NewNamespace(ns),
		NetworkPolicy: NewNetworkPolicy(client.ObjectKey{Namespace: ns, Name: t.Key.Name}),
	}
}

func ConfigureTenantDeps(ctx context.Context, td TenantDeps) {
	td.configure(ctx)
}

func ApplyTenantDeps(ctx context.Context, cl client.Client, t *Tenant) (TenantDeps, error) {
	td := NewTenantDeps(t)

	if _, err := td.Load(ctx, cl); err != nil {
		return nil, err
	}

	ConfigureTenantDeps(ctx, td)

	if err := td.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return td, nil
}
