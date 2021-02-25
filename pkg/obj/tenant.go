package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	TenantStatusReasonNamespaceReady = "NamespaceReady"
	TenantStatusReasonNamespaceError = "NamespaceError"

	TenantStatusReasonEventSinkMissing       = "EventSinkMissing"
	TenantStatusReasonEventSinkNotConfigured = "EventSinkNotConfigured"
	TenantStatusReasonEventSinkReady         = "EventSinkReady"

	TenantStatusReasonToolInjectionNotDefined = "ToolInjectionNotDefined"
	TenantStatusReasonToolInjectionError      = "ToolInjectionError"

	TenantStatusReasonReady = "Ready"
	TenantStatusReasonError = "Error"
)

var (
	TenantKind = relayv1beta1.SchemeGroupVersion.WithKind("Tenant")
)

type Tenant struct {
	Key    client.ObjectKey
	Object *relayv1beta1.Tenant
}

var _ lifecycle.Finalizable = &Tenant{}
var _ lifecycle.LabelAnnotatableFrom = &Tenant{}
var _ lifecycle.Loader = &Tenant{}
var _ lifecycle.Persister = &Tenant{}

func (t *Tenant) Finalizing() bool {
	return !t.Object.GetDeletionTimestamp().IsZero()
}

func (t *Tenant) AddFinalizer(ctx context.Context, name string) bool {
	return helper.AddFinalizer(&t.Object.ObjectMeta, name)
}

func (t *Tenant) RemoveFinalizer(ctx context.Context, name string) bool {
	return helper.RemoveFinalizer(&t.Object.ObjectMeta, name)
}

func (t *Tenant) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	helper.CopyLabelsAndAnnotations(&t.Object.ObjectMeta, from)
}

func (t *Tenant) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, t.Object, helper.WithObjectKey(t.Key))
}

func (t *Tenant) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, t.Object)
}

func (t *Tenant) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, t.Key, t.Object)
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
	return &Tenant{
		Key:    key,
		Object: &relayv1beta1.Tenant{},
	}
}

func NewTenantPatcher(upd, orig *Tenant) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
