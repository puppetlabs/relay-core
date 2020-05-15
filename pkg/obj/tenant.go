package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	TenantKind = relayv1beta1.SchemeGroupVersion.WithKind("Tenant")
)

type Tenant struct {
	Key    client.ObjectKey
	Object *relayv1beta1.Tenant
}

var _ Loader = &Tenant{}
var _ Persister = &Tenant{}
var _ Finalizable = &Tenant{}

func (t *Tenant) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, t.Key, t.Object)
}

func (t *Tenant) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, t.Object)
}

func (t *Tenant) Finalizing() bool {
	return !t.Object.GetDeletionTimestamp().IsZero()
}

func (t *Tenant) AddFinalizer(ctx context.Context, name string) bool {
	return AddFinalizer(&t.Object.ObjectMeta, name)
}

func (t *Tenant) RemoveFinalizer(ctx context.Context, name string) bool {
	return RemoveFinalizer(&t.Object.ObjectMeta, name)
}

func (t *Tenant) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, t.Key, t.Object)
}

func (t *Tenant) Own(ctx context.Context, other Ownable) error {
	return other.Owned(ctx, Owner{GVK: TenantKind, Object: t.Object})
}

func (t *Tenant) Managed() bool {
	return t.Object.Spec.NamespaceTemplate.Metadata.GetName() != ""
}

func NewTenant(key client.ObjectKey) *Tenant {
	return &Tenant{
		Key:    key,
		Object: &relayv1beta1.Tenant{},
	}
}
