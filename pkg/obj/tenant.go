package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TenantStatusReasonNamespaceReady = "NamespaceReady"
	TenantStatusReasonNamespaceError = "NamespaceError"

	TenantStatusReasonEventSinkMissing       = "EventSinkMissing"
	TenantStatusReasonEventSinkNotConfigured = "EventSinkNotConfigured"
	TenantStatusReasonEventSinkReady         = "EventSinkReady"

	TenantStatusReasonReady = "Ready"
	TenantStatusReasonError = "Error"
)

type Tenant struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *relayv1beta1.Tenant
}

func makeTenant(key client.ObjectKey, obj *relayv1beta1.Tenant) *Tenant {
	t := &Tenant{Key: key, Object: obj}
	t.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&t.Key, lifecycle.TypedObject{GVK: relayv1beta1.TenantKind, Object: t.Object})
	return t
}

func (t *Tenant) Copy() *Tenant {
	return makeTenant(t.Key, t.Object.DeepCopy())
}

func (t *Tenant) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, t.Object)
}

func (t *Tenant) Managed() bool {
	return t.Object.Spec.NamespaceTemplate.Metadata.GetName() != ""
}

func (t *Tenant) Ready() bool {
	for _, cond := range t.Object.Status.Conditions {
		if cond.Type != relayv1beta1.TenantReady {
			continue
		}

		return cond.Status == corev1.ConditionTrue
	}

	return false
}

func NewTenant(key client.ObjectKey) *Tenant {
	return makeTenant(key, &relayv1beta1.Tenant{})
}

func NewTenantFromObject(obj *relayv1beta1.Tenant) *Tenant {
	return makeTenant(client.ObjectKeyFromObject(obj), obj)
}

func NewTenantPatcher(upd, orig *Tenant) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
