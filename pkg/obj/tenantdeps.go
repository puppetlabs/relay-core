package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APITriggerEventSink struct {
	Sink        *relayv1beta1.APITriggerEventSink
	TokenSecret *OpaqueSecret
}

var _ Loader = &APITriggerEventSink{}

func (tes *APITriggerEventSink) Load(ctx context.Context, cl client.Client) (bool, error) {
	return IgnoreNilLoader{tes.TokenSecret}.Load(ctx, cl)
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
		tes.TokenSecret = NewOpaqueSecret(client.ObjectKey{
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
	Tenant *Tenant

	// StaleNamespace is the old namespace of a tenant that needs to be cleaned
	// up.
	StaleNamespace *Namespace

	Namespace     *Namespace
	NetworkPolicy *NetworkPolicy
	LimitRange    *LimitRange

	APITriggerEventSink *APITriggerEventSink
	ToolInjection       *ToolInjection
}

var _ Persister = &TenantDeps{}
var _ Loader = &TenantDeps{}
var _ Deleter = &TenantDeps{}

func (td *TenantDeps) Persist(ctx context.Context, cl client.Client) error {
	if !td.Tenant.Managed() {
		return nil
	}

	ps := []Persister{
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

func (td *TenantDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	loaders := Loaders{IgnoreNilLoader{td.APITriggerEventSink}}

	if !td.Tenant.Managed() {
		loaders = append(loaders, RequiredLoader{td.Namespace})
	} else {
		loaders = append(loaders, td.Namespace, td.NetworkPolicy, td.LimitRange)
	}

	// Check for stale namespace. We only clean up the stale namespace if it was
	// managed.
	if td.Tenant.Object.Status.Namespace != "" && td.Tenant.Object.Status.Namespace != td.Tenant.Key.Namespace && td.Tenant.Object.Status.Namespace != td.Namespace.Name {
		td.StaleNamespace = NewNamespace(td.Tenant.Object.Status.Namespace)
		loaders = append(loaders, td.StaleNamespace)
	}

	return loaders.Load(ctx, cl)
}

func (td *TenantDeps) Delete(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := td.DeleteStale(ctx, cl); err != nil {
		return false, err
	}

	if !td.Tenant.Managed() {
		return true, nil
	}

	if td.Namespace.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := IsDependencyOf(td.Namespace.Object.ObjectMeta, Owner{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind}); err != nil {
		return false, err
	} else if ok {
		return td.Namespace.Delete(ctx, cl)
	}

	return true, nil
}

func (td *TenantDeps) DeleteStale(ctx context.Context, cl client.Client) (bool, error) {
	if td.StaleNamespace == nil || td.StaleNamespace.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := IsDependencyOf(td.StaleNamespace.Object.ObjectMeta, Owner{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind}); err != nil {
		return false, err
	} else if ok {
		return td.StaleNamespace.Delete(ctx, cl)
	}

	return true, nil
}

func NewTenantDeps(t *Tenant) *TenantDeps {
	td := &TenantDeps{
		Tenant: t,
	}

	if !t.Managed() {
		td.Namespace = NewNamespace(t.Key.Namespace)
	} else {
		ns := t.Object.Spec.NamespaceTemplate.Metadata.GetName()

		td.Namespace = NewNamespace(ns)
		td.NetworkPolicy = NewNetworkPolicy(client.ObjectKey{Namespace: ns, Name: t.Key.Name})
		td.LimitRange = NewLimitRange(client.ObjectKey{Namespace: ns, Name: t.Key.Name})
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

	SetDependencyOf(&td.Namespace.Object.ObjectMeta, Owner{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind})
	SetDependencyOf(&td.NetworkPolicy.Object.ObjectMeta, Owner{Object: td.Tenant.Object, GVK: relayv1beta1.TenantKind})

	td.Namespace.Label(ctx, model.RelayControllerTenantWorkloadLabel, "true")
	td.Namespace.LabelAnnotateFrom(ctx, td.Tenant.Object.Spec.NamespaceTemplate.Metadata)

	ConfigureNetworkPolicyForTenant(td.NetworkPolicy)
	ConfigureLimitRange(td.LimitRange)
}

func ApplyTenantDeps(ctx context.Context, cl client.Client, t *Tenant) (*TenantDeps, error) {
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
