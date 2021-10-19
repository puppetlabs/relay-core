package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/jsonutil/pkg/types"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	networkingv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/networkingv1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	pvpoolv1alpha1obj "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1/obj"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkflowRunDepsLoadResult struct {
	Upstream bool
	All      bool
}

// WorkflowRunDeps represents the dependencies of a WorkflowRun.
type WorkflowRunDeps struct {
	WorkflowRun  *obj.WorkflowRun
	Workflow     *obj.Workflow
	WorkflowDeps *WorkflowDeps

	Standalone bool

	Issuer authenticate.Issuer

	ToolInjectionPoolRef pvpoolv1alpha1.PoolReference

	OwnerConfigMap *corev1obj.ConfigMap

	NetworkPolicy *networkingv1obj.NetworkPolicy

	ToolInjectionCheckout *PoolRefPredicatedCheckout

	ImmutableConfigMap *corev1obj.ConfigMap
	MutableConfigMap   *corev1obj.ConfigMap

	MetadataAPIURL                        *url.URL
	MetadataAPIServiceAccount             *corev1obj.ServiceAccount
	MetadataAPIServiceAccountTokenSecrets *corev1obj.ServiceAccountTokenSecrets
	MetadataAPIRole                       *rbacv1obj.Role
	MetadataAPIRoleBinding                *rbacv1obj.RoleBinding

	PipelineServiceAccount  *corev1obj.ServiceAccount
	UntrustedServiceAccount *corev1obj.ServiceAccount
}

var _ lifecycle.Deleter = &WebhookTriggerDeps{}
var _ lifecycle.Persister = &WebhookTriggerDeps{}

func (wrd *WorkflowRunDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if wrd.OwnerConfigMap == nil || wrd.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(
		wrd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: wrd.WorkflowRun.Object,
			GVK:    nebulav1.WorkflowRunKind,
		}); err != nil {
		return false, err
	} else if ok {
		return wrd.OwnerConfigMap.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func (wrd *WorkflowRunDeps) Load(ctx context.Context, cl client.Client) (*WorkflowRunDepsLoadResult, error) {
	if ok, err := wrd.Workflow.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		return &WorkflowRunDepsLoadResult{}, nil
	}

	wrd.WorkflowDeps = NewWorkflowDeps(wrd.Workflow)

	if lr, err := wrd.WorkflowDeps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !lr.All {
		return &WorkflowRunDepsLoadResult{}, nil
	}

	key := client.ObjectKey{
		Namespace: wrd.WorkflowDeps.TenantDeps.Namespace.Name,
		Name:      wrd.WorkflowRun.Key.Name,
	}

	wrd.OwnerConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "owner"))

	wrd.NetworkPolicy = networkingv1obj.NewNetworkPolicy(key)

	wrd.ToolInjectionCheckout = &PoolRefPredicatedCheckout{
		Checkout: pvpoolv1alpha1obj.NewCheckout(
			SuffixObjectKeyWithHashOfObjectKey(SuffixObjectKey(key, "tools"),
				client.ObjectKey{
					Namespace: wrd.ToolInjectionPoolRef.Namespace,
					Name:      wrd.ToolInjectionPoolRef.Name,
				},
			),
		),
	}

	wrd.ImmutableConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "immutable"))
	wrd.MutableConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "mutable"))

	wrd.MetadataAPIServiceAccount = corev1obj.NewServiceAccount(SuffixObjectKey(key, "metadata-api"))
	wrd.MetadataAPIRole = rbacv1obj.NewRole(SuffixObjectKey(key, "metadata-api"))
	wrd.MetadataAPIRoleBinding = rbacv1obj.NewRoleBinding(SuffixObjectKey(key, "metadata-api"))

	wrd.PipelineServiceAccount = corev1obj.NewServiceAccount(SuffixObjectKey(key, "pipeline"))
	wrd.UntrustedServiceAccount = corev1obj.NewServiceAccount(SuffixObjectKey(key, "untrusted"))

	wrd.MetadataAPIServiceAccountTokenSecrets = corev1obj.NewServiceAccountTokenSecrets(wrd.MetadataAPIServiceAccount)

	ok, err := lifecycle.Loaders{
		wrd.OwnerConfigMap,
		lifecycle.IgnoreNilLoader{Loader: wrd.NetworkPolicy},
		wrd.ToolInjectionCheckout,
		wrd.ImmutableConfigMap,
		wrd.MutableConfigMap,
		wrd.MetadataAPIServiceAccount,
		lifecycle.NewPrereqLoader(
			corev1obj.NewServiceAccountTokenSecretsDefaultPresentPoller(wrd.MetadataAPIServiceAccountTokenSecrets),
			wrd.MetadataAPIServiceAccount.Object,
		),
		wrd.MetadataAPIRole,
		wrd.MetadataAPIRoleBinding,
		wrd.PipelineServiceAccount,
		wrd.UntrustedServiceAccount,
	}.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	return &WorkflowRunDepsLoadResult{
		Upstream: true,
		All:      ok,
	}, nil
}

func (wrd *WorkflowRunDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := wrd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		wrd.ToolInjectionCheckout,
		wrd.ImmutableConfigMap,
		wrd.MutableConfigMap,
		wrd.MetadataAPIServiceAccount,
		wrd.MetadataAPIRole,
		wrd.MetadataAPIRoleBinding,
		wrd.PipelineServiceAccount,
		wrd.UntrustedServiceAccount,
	}
	for _, o := range os {
		if err := wrd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	if wrd.NetworkPolicy != nil {
		if err := wrd.OwnerConfigMap.Own(ctx, wrd.NetworkPolicy); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		lifecycle.IgnoreNilPersister{Persister: wrd.NetworkPolicy},
		wrd.ToolInjectionCheckout,
		wrd.ImmutableConfigMap,
		wrd.MutableConfigMap,
		wrd.MetadataAPIServiceAccount,
		wrd.MetadataAPIRole,
		wrd.MetadataAPIRoleBinding,
		wrd.PipelineServiceAccount,
		wrd.UntrustedServiceAccount,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	// Sync token secrets.
	if _, err := corev1obj.NewServiceAccountTokenSecretsDefaultPresentPoller(wrd.MetadataAPIServiceAccountTokenSecrets).Load(ctx, cl); err != nil {
		return err
	}

	return nil
}

func (wrd *WorkflowRunDeps) AnnotateStepToken(ctx context.Context, target *metav1.ObjectMeta, ws *relayv1beta1.Step) error {
	if _, found := target.Annotations[authenticate.KubernetesTokenAnnotation]; found {
		// We only add this once and exactly once per run per target.
		return nil
	}

	ms := ModelStep(wrd.WorkflowRun, ws)
	now := time.Now()

	// FIXME Temporarily avoid unknown transient issue
	if wrd.MetadataAPIServiceAccountTokenSecrets == nil ||
		wrd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret == nil ||
		wrd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Object == nil {
		return errors.New("no default token secret for workflow run")
	}

	sat, err := wrd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Token()
	if err != nil {
		return errmark.MarkTransientIf(err, errmark.RuleIs(corev1obj.ErrServiceAccountTokenMissingData))
	}

	annotations := wrd.WorkflowRun.Object.GetAnnotations()

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Issuer:    authenticate.ControllerIssuer,
			Audience:  jwt.Audience{authenticate.MetadataAPIAudienceV1},
			Subject:   path.Join(ms.Type().Plural, ms.Hash().HexEncoding()),
			Expiry:    jwt.NewNumericDate(now.Add(1*time.Hour + 5*time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},

		KubernetesNamespaceName:       wrd.WorkflowDeps.TenantDeps.Namespace.Name,
		KubernetesNamespaceUID:        string(wrd.WorkflowDeps.TenantDeps.Namespace.Object.GetUID()),
		KubernetesServiceAccountToken: sat,

		RelayDomainID: annotations[model.RelayDomainIDAnnotation],
		RelayTenantID: annotations[model.RelayTenantIDAnnotation],
		RelayRunID:    ms.Run.ID,
		RelayName:     ms.Name,

		RelayKubernetesImmutableConfigMapName: wrd.ImmutableConfigMap.Key.Name,
		RelayKubernetesMutableConfigMapName:   wrd.MutableConfigMap.Key.Name,

		RelayVaultEnginePath:     annotations[model.RelayVaultEngineMountAnnotation],
		RelayVaultSecretPath:     annotations[model.RelayVaultSecretPathAnnotation],
		RelayVaultConnectionPath: annotations[model.RelayVaultConnectionPathAnnotation],
	}

	td := wrd.WorkflowDeps.TenantDeps
	if sink := td.APIWorkflowExecutionSink; sink != nil {
		if u, _ := url.Parse(sink.URL()); u != nil {
			claims.RelayWorkflowExecutionAPIURL = &types.URL{URL: u}
			claims.RelayWorkflowExecutionAPIToken, _ = sink.Token()
		}
	}

	tok, err := wrd.Issuer.Issue(ctx, claims)
	if err != nil {
		return err
	}

	helper.Annotate(target, authenticate.KubernetesTokenAnnotation, string(tok))
	helper.Annotate(target, authenticate.KubernetesSubjectAnnotation, claims.Subject)

	return nil
}

type WorkflowRunDepsOption func(wrd *WorkflowRunDeps)

func WorkflowRunDepsWithStandaloneMode(standalone bool) WorkflowRunDepsOption {
	return func(wrd *WorkflowRunDeps) {
		if standalone {
			wrd.Standalone = true
		}
	}
}

func WorkflowRunDepsWithToolInjectionPool(pr pvpoolv1alpha1.PoolReference) WorkflowRunDepsOption {
	return func(wrd *WorkflowRunDeps) {
		wrd.ToolInjectionPoolRef = pr
	}
}

func NewWorkflowRunDeps(wr *obj.WorkflowRun, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WorkflowRunDepsOption) *WorkflowRunDeps {
	wrd := &WorkflowRunDeps{
		WorkflowRun: wr,
		Workflow: obj.NewWorkflow(client.ObjectKey{
			Namespace: wr.Key.Namespace,
			Name:      wr.Object.Spec.WorkflowRef.Name,
		}),

		Issuer: issuer,

		MetadataAPIURL: metadataAPIURL,
	}

	for _, opt := range opts {
		opt(wrd)
	}

	return wrd
}

func ConfigureWorkflowRunDeps(ctx context.Context, wrd *WorkflowRunDeps) error {
	if err := DependencyManager.SetDependencyOf(
		wrd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: wrd.WorkflowRun.Object,
			GVK:    nebulav1.WorkflowRunKind,
		}); err != nil {
		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		wrd.ToolInjectionCheckout,
		wrd.ImmutableConfigMap,
		wrd.MutableConfigMap,
		wrd.MetadataAPIServiceAccount,
		wrd.MetadataAPIRole,
		wrd.PipelineServiceAccount,
		wrd.UntrustedServiceAccount,
	}
	for _, laf := range lafs {
		laf.LabelAnnotateFrom(ctx, wrd.WorkflowRun.Object)
		lifecycle.Label(ctx, laf, model.RelayControllerWorkflowRunIDLabel, wrd.WorkflowRun.Key.Name)
	}

	if wrd.Standalone {
		wrd.NetworkPolicy.AllowAll()
	} else {
		ConfigureNetworkPolicyForWorkflowRun(wrd.NetworkPolicy, wrd.WorkflowRun)
	}

	ConfigureToolInjectionCheckout(wrd.ToolInjectionCheckout, wrd.WorkflowDeps.TenantDeps.Tenant, wrd.ToolInjectionPoolRef)

	if err := ConfigureImmutableConfigMapForWorkflowRun(ctx, wrd.ImmutableConfigMap, wrd); err != nil {
		return err
	}
	if err := ConfigureMutableConfigMapForWorkflowRun(ctx, wrd.MutableConfigMap, wrd.WorkflowRun); err != nil {
		return err
	}

	ConfigureMetadataAPIServiceAccount(wrd.MetadataAPIServiceAccount)
	ConfigureMetadataAPIRole(wrd.MetadataAPIRole, wrd.ImmutableConfigMap, wrd.MutableConfigMap)
	ConfigureMetadataAPIRoleBinding(wrd.MetadataAPIRoleBinding, wrd.MetadataAPIServiceAccount, wrd.MetadataAPIRole)
	ConfigureUntrustedServiceAccount(wrd.PipelineServiceAccount)
	ConfigureUntrustedServiceAccount(wrd.UntrustedServiceAccount)

	return nil
}

func ApplyWorkflowRunDeps(ctx context.Context, cl client.Client, wr *obj.WorkflowRun, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WorkflowRunDepsOption) (*WorkflowRunDeps, error) {
	deps := NewWorkflowRunDeps(wr, issuer, metadataAPIURL, opts...)

	if lr, err := deps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !lr.Upstream {
		return nil, fmt.Errorf("waiting on WorkflowRun upstream dependencies")
	}

	if err := ConfigureWorkflowRunDeps(ctx, deps); err != nil {
		return nil, err
	}

	if err := deps.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return deps, nil
}
