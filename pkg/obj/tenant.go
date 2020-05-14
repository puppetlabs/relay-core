package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (t *Tenant) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, t.Object)
}

func (t *Tenant) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, t.Key, t.Object)
}

func (t *Tenant) Own(ctx context.Context, other Ownable) {
	other.Owned(ctx, metav1.NewControllerRef(t.Object, TenantKind))
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
