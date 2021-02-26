package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	networkingv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/networkingv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APITriggerEventSink struct {
	Sink        *relayv1beta1.APITriggerEventSink
	TokenSecret *corev1obj.OpaqueSecret
}

var _ lifecycle.Loader = &APITriggerEventSink{}

func (tes *APITriggerEventSink) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.IgnoreNilLoader{tes.TokenSecret}.Load(ctx, cl)
}

func (tes *APITriggerEventSink) URL() string {
	return tes.Sink.URL
}

func (tes *APITriggerEventSink) Token() (string, bool) {
	if tes.Sink.Token != "" {
		return tes.Sink.Token, true
	} else if tes.TokenSecret != nil {
		return tes.TokenSecret.Data(tes.Sink.TokenFrom.SecretKeyRef.Key)
	}

	return "", false
}

func NewAPITriggerEventSink(namespace string, sink *relayv1beta1.APITriggerEventSink) *APITriggerEventSink {
	tes := &APITriggerEventSink{
		Sink: sink,
	}

	if sink.TokenFrom != nil && sink.TokenFrom.SecretKeyRef != nil {
		tes.TokenSecret = corev1obj.NewOpaqueSecret(client.ObjectKey{
			Namespace: namespace,
			Name:      sink.TokenFrom.SecretKeyRef.Name,
		})
	}

	return tes
}

type ToolInjection struct {
	VolumeClaimTemplate *corev1.PersistentVolumeClaim
}

func NewToolInjection(namespace string, toolInjection relayv1beta1.ToolInjection) *ToolInjection {
	ti := &ToolInjection{
		VolumeClaimTemplate: toolInjection.VolumeClaimTemplate,
	}

	return ti
}

type TenantDeps struct {
	Tenant *obj.Tenant

	// StaleNamespace is the old namespace of a tenant that needs to be cleaned
	// up.
	StaleNamespace *corev1obj.Namespace

	Namespace     *corev1obj.Namespace
	NetworkPolicy *networkingv1obj.NetworkPolicy
	LimitRange    *corev1obj.LimitRange

	APITriggerEventSink *APITriggerEventSink
	ToolInjection       *ToolInjection
}

var _ lifecycle.Deleter = &TenantDeps{}
var _ lifecycle.Loader = &TenantDeps{}
var _ lifecycle.Persister = &TenantDeps{}

func (td *TenantDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if _, err := td.DeleteStale(ctx, cl, opts...); err != nil {
		return false, err
	}

	if !td.Tenant.Managed() {
		return true, nil
	}

	if td.Namespace.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(td.Namespace.Object, lifecycle.TypedObject{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind}); err != nil {
		return false, err
	} else if ok {
		return td.Namespace.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func (td *TenantDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	loaders := lifecycle.Loaders{lifecycle.IgnoreNilLoader{td.APITriggerEventSink}}

	if !td.Tenant.Managed() {
		loaders = append(loaders, lifecycle.RequiredLoader{td.Namespace})
	} else {
		loaders = append(loaders, td.Namespace, td.NetworkPolicy, td.LimitRange)
	}

	// Check for stale namespace. We only clean up the stale namespace if it was
	// managed.
	if td.Tenant.Object.Status.Namespace != "" && td.Tenant.Object.Status.Namespace != td.Tenant.Key.Namespace && td.Tenant.Object.Status.Namespace != td.Namespace.Name {
		td.StaleNamespace = corev1obj.NewNamespace(td.Tenant.Object.Status.Namespace)
		loaders = append(loaders, td.StaleNamespace)
	}

	return loaders.Load(ctx, cl)
}

func (td *TenantDeps) Persist(ctx context.Context, cl client.Client) error {
	if !td.Tenant.Managed() {
		return nil
	}

	ps := []lifecycle.Persister{
		td.Namespace,
		td.NetworkPolicy,
		td.LimitRange,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (td *TenantDeps) DeleteStale(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if td.StaleNamespace == nil || td.StaleNamespace.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(td.StaleNamespace.Object, lifecycle.TypedObject{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind}); err != nil {
		return false, err
	} else if ok {
		return td.StaleNamespace.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func NewTenantDeps(t *obj.Tenant) *TenantDeps {
	td := &TenantDeps{
		Tenant: t,
	}

	if !t.Managed() {
		td.Namespace = corev1obj.NewNamespace(t.Key.Namespace)
	} else {
		ns := t.Object.Spec.NamespaceTemplate.Metadata.GetName()

		td.Namespace = corev1obj.NewNamespace(ns)
		td.NetworkPolicy = networkingv1obj.NewNetworkPolicy(client.ObjectKey{Namespace: ns, Name: t.Key.Name})
		td.LimitRange = corev1obj.NewLimitRange(client.ObjectKey{Namespace: ns, Name: t.Key.Name})
	}

	if sink := t.Object.Spec.TriggerEventSink.API; sink != nil {
		td.APITriggerEventSink = NewAPITriggerEventSink(td.Tenant.Key.Namespace, sink)
	}

	td.ToolInjection = NewToolInjection(td.Tenant.Key.Namespace, td.Tenant.Object.Spec.ToolInjection)

	return td
}

func ConfigureTenantDeps(ctx context.Context, td *TenantDeps) {
	if !td.Tenant.Managed() {
		return
	}

	DependencyManager.SetDependencyOf(&td.Namespace.Object.ObjectMeta, lifecycle.TypedObject{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind})
	DependencyManager.SetDependencyOf(&td.NetworkPolicy.Object.ObjectMeta, lifecycle.TypedObject{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind})

	lifecycle.Label(ctx, td.Namespace, model.RelayControllerTenantWorkloadLabel, "true")
	td.Namespace.LabelAnnotateFrom(ctx, &td.Tenant.Object.Spec.NamespaceTemplate.Metadata)

	ConfigureNetworkPolicyForTenant(td.NetworkPolicy)
	ConfigureLimitRange(td.LimitRange)
}

func ApplyTenantDeps(ctx context.Context, cl client.Client, t *obj.Tenant) (*TenantDeps, error) {
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

type TenantDepsResult struct {
	TenantDeps *TenantDeps
	Error      error
}

func AsTenantDepsResult(td *TenantDeps, err error) *TenantDepsResult {
	return &TenantDepsResult{
		TenantDeps: td,
		Error:      err,
	}
}
