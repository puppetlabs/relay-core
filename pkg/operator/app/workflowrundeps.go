package app

import (
	"context"
	"net/url"
	"path"
	"time"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	networkingv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/networkingv1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkflowRunDeps represents the Kubernetes objects required to create a Pipeline.
type WorkflowRunDeps struct {
	WorkflowRun *obj.WorkflowRun
	Issuer      authenticate.Issuer

	Namespace *corev1obj.Namespace

	// TODO: This belongs at the Tenant as it should apply to the whole
	// namespace.
	LimitRange *corev1obj.LimitRange

	NetworkPolicy *networkingv1obj.NetworkPolicy

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

var _ lifecycle.Loader = &WorkflowRunDeps{}
var _ lifecycle.Persister = &WorkflowRunDeps{}

func (wrd *WorkflowRunDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.Loaders{
		lifecycle.RequiredLoader{wrd.Namespace},
		lifecycle.IgnoreNilLoader{wrd.LimitRange},
		lifecycle.IgnoreNilLoader{wrd.NetworkPolicy},
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
}

func (wrd *WorkflowRunDeps) Persist(ctx context.Context, cl client.Client) error {
	ps := []lifecycle.Persister{
		lifecycle.IgnoreNilPersister{wrd.LimitRange},
		lifecycle.IgnoreNilPersister{wrd.NetworkPolicy},
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

func (wrd *WorkflowRunDeps) AnnotateStepToken(ctx context.Context, target *metav1.ObjectMeta, ws *nebulav1.WorkflowStep) error {
	if _, found := target.Annotations[authenticate.KubernetesTokenAnnotation]; found {
		// We only add this once and exactly once per run per target.
		return nil
	}

	ms := ModelStep(wrd.WorkflowRun, ws)
	now := time.Now()

	sat, err := wrd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Token()
	if err != nil {
		return err
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
			wrd.LimitRange = nil
		}
	}
}

func NewWorkflowRunDeps(wr *obj.WorkflowRun, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WorkflowRunDepsOption) *WorkflowRunDeps {
	key := wr.Key

	wrd := &WorkflowRunDeps{
		WorkflowRun: wr,
		Issuer:      issuer,

		Namespace: corev1obj.NewNamespace(key.Namespace),

		LimitRange: corev1obj.NewLimitRange(key),

		NetworkPolicy: networkingv1obj.NewNetworkPolicy(key),

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

	if wrd.LimitRange != nil {
		if err := wrd.WorkflowRun.Own(ctx, wrd.LimitRange); err != nil {
			return err
		}
	}

	if wrd.NetworkPolicy != nil {
		if err := wrd.WorkflowRun.Own(ctx, wrd.NetworkPolicy); err != nil {
			return err
		}
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
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

	if wrd.LimitRange != nil {
		ConfigureLimitRange(wrd.LimitRange)
	}

	if wrd.NetworkPolicy != nil {
		ConfigureNetworkPolicyForWorkflowRun(wrd.NetworkPolicy, wrd.WorkflowRun)
	}

	if err := ConfigureImmutableConfigMapForWorkflowRun(ctx, wrd.ImmutableConfigMap, wrd.WorkflowRun); err != nil {
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

	if _, err := deps.Load(ctx, cl); err != nil {
		return nil, err
	}

	if err := ConfigureWorkflowRunDeps(ctx, deps); err != nil {
		return nil, err
	}

	if err := deps.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return deps, nil
}
