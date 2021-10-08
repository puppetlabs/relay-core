package app

import (
	"context"
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

	ToolInjectionPoolRef pvpoolv1alpha1.PoolReference
	Issuer               authenticate.Issuer

	Namespace *corev1obj.Namespace

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

var _ lifecycle.Persister = &WorkflowRunDeps{}

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

	ok, err := lifecycle.Loaders{
		lifecycle.RequiredLoader{Loader: wrd.Namespace},
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

		KubernetesNamespaceName:       wrd.Namespace.Name,
		KubernetesNamespaceUID:        string(wrd.Namespace.Object.GetUID()),
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
			wrd.NetworkPolicy = nil
		}
	}
}

func WorkflowRunDepsWithToolInjectionPool(pr pvpoolv1alpha1.PoolReference) WorkflowRunDepsOption {
	return func(wrd *WorkflowRunDeps) {
		wrd.ToolInjectionPoolRef = pr
	}
}

func NewWorkflowRunDeps(wr *obj.WorkflowRun, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WorkflowRunDepsOption) *WorkflowRunDeps {
	key := wr.Key

	wrd := &WorkflowRunDeps{
		WorkflowRun: wr,
		Workflow: obj.NewWorkflow(client.ObjectKey{
			Namespace: key.Namespace,
			Name:      wr.Object.Spec.WorkflowRef.Name,
		}),

		Issuer: issuer,

		Namespace: corev1obj.NewNamespace(key.Namespace),

		NetworkPolicy: networkingv1obj.NewNetworkPolicy(key),

		ToolInjectionCheckout: &PoolRefPredicatedCheckout{
			Checkout: pvpoolv1alpha1obj.NewCheckout(SuffixObjectKey(key, "tools")),
		},

		ImmutableConfigMap: corev1obj.NewConfigMap(SuffixObjectKey(key, "immutable")),
		MutableConfigMap:   corev1obj.NewConfigMap(SuffixObjectKey(key, "mutable")),

		MetadataAPIURL:            metadataAPIURL,
		MetadataAPIServiceAccount: corev1obj.NewServiceAccount(SuffixObjectKey(key, "metadata-api")),
		MetadataAPIRole:           rbacv1obj.NewRole(SuffixObjectKey(key, "metadata-api")),
		MetadataAPIRoleBinding:    rbacv1obj.NewRoleBinding(SuffixObjectKey(key, "metadata-api")),

		PipelineServiceAccount:  corev1obj.NewServiceAccount(SuffixObjectKey(key, "pipeline")),
		UntrustedServiceAccount: corev1obj.NewServiceAccount(SuffixObjectKey(key, "untrusted")),
	}
	wrd.MetadataAPIServiceAccountTokenSecrets = corev1obj.NewServiceAccountTokenSecrets(wrd.MetadataAPIServiceAccount)

	for _, opt := range opts {
		opt(wrd)
	}

	return wrd
}

func ConfigureWorkflowRunDeps(ctx context.Context, wrd *WorkflowRunDeps) error {
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
		if err := wrd.WorkflowRun.Own(ctx, o); err != nil {
			return err
		}
	}

	if wrd.NetworkPolicy != nil {
		if err := wrd.WorkflowRun.Own(ctx, wrd.NetworkPolicy); err != nil {
			return err
		}
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
	}

	if wrd.NetworkPolicy != nil {
		ConfigureNetworkPolicyForWorkflowRun(wrd.NetworkPolicy, wrd.WorkflowRun)
	}

	ConfigureToolInjectionCheckoutForWorkflowRun(wrd.ToolInjectionCheckout,
		wrd.WorkflowRun, wrd.WorkflowDeps.TenantDeps.Tenant, wrd.ToolInjectionPoolRef)

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
