package obj

import (
	"context"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type APITriggerEventSink struct {
	Sink        *relayv1beta1.APITriggerEventSink
	TokenSecret *OpaqueSecret
}

var _ Loader = &APITriggerEventSink{}

func (tes *APITriggerEventSink) Load(ctx context.Context, cl client.Client) (bool, error) {
	return RequiredLoader{IgnoreNilLoader{tes.TokenSecret}}.Load(ctx, cl)
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

type TenantDeps struct {
	Tenant *Tenant

	Namespace     *Namespace
	NetworkPolicy *NetworkPolicy

	APITriggerEventSink *APITriggerEventSink
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
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (td *TenantDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	loaders := Loaders{RequiredLoader{IgnoreNilLoader{td.APITriggerEventSink}}}

	if !td.Tenant.Managed() {
		loaders = append(loaders, RequiredLoader{td.Namespace})
	} else {
		loaders = append(loaders, td.Namespace, td.NetworkPolicy)
	}

	return loaders.Load(ctx, cl)
}

func (td *TenantDeps) Delete(ctx context.Context, cl client.Client) (bool, error) {
	if !td.Tenant.Managed() {
		return true, nil
	}

	if DependencyOf(td.Namespace.Object.ObjectMeta, td.Tenant.Object.ObjectMeta) {
		return td.Namespace.Delete(ctx, cl)
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
	}

	if sink := t.Object.Spec.TriggerEventSink.API; sink != nil {
		td.APITriggerEventSink = NewAPITriggerEventSink(td.Namespace.Name, sink)
	}

	return td
}

func ConfigureTenantDeps(ctx context.Context, td *TenantDeps) {
	if !td.Tenant.Managed() {
		return
	}

	SetDependencyOf(&td.Namespace.Object.ObjectMeta, td.Tenant.Object.ObjectMeta)
	SetDependencyOf(&td.NetworkPolicy.Object.ObjectMeta, td.Tenant.Object.ObjectMeta)

	td.Namespace.Label(ctx, model.RelayControllerTenantWorkloadLabel, "true")
	td.Namespace.LabelAnnotateFrom(ctx, td.Tenant.Object.Spec.NamespaceTemplate.Metadata)

	ConfigureNetworkPolicyForTenant(td.NetworkPolicy)
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
