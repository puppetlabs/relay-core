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
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RunDepsLoadResult struct {
	Upstream bool
	All      bool
}

// RunDeps represents the dependencies of a Run.
type RunDeps struct {
	Run          *obj.Run
	Workflow     *obj.Workflow
	WorkflowDeps *WorkflowDeps

	Standalone bool

	Issuer authenticate.Issuer

	OwnerConfigMap *corev1obj.ConfigMap

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

var _ lifecycle.Deleter = &WebhookTriggerDeps{}
var _ lifecycle.Persister = &WebhookTriggerDeps{}

func (rd *RunDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if rd.OwnerConfigMap == nil || rd.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(
		rd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: rd.Run.Object,
			GVK:    relayv1beta1.RunKind,
		}); err != nil {
		return false, err
	} else if ok {
		return rd.OwnerConfigMap.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func (rd *RunDeps) Load(ctx context.Context, cl client.Client) (*RunDepsLoadResult, error) {
	if ok, err := rd.Workflow.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		return &RunDepsLoadResult{}, nil
	}

	rd.WorkflowDeps = NewWorkflowDeps(rd.Workflow)

	if lr, err := rd.WorkflowDeps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !lr.All {
		return &RunDepsLoadResult{}, nil
	}

	key := client.ObjectKey{
		Namespace: rd.WorkflowDeps.TenantDeps.Namespace.Name,
		Name:      rd.Run.Key.Name,
	}

	rd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	rd.NetworkPolicy = networkingv1obj.NewNetworkPolicy(key)

	rd.ImmutableConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "immutable"))
	rd.MutableConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "mutable"))

	rd.MetadataAPIServiceAccount = corev1obj.NewServiceAccount(helper.SuffixObjectKey(key, "metadata-api"))
	rd.MetadataAPIRole = rbacv1obj.NewRole(helper.SuffixObjectKey(key, "metadata-api"))
	rd.MetadataAPIRoleBinding = rbacv1obj.NewRoleBinding(helper.SuffixObjectKey(key, "metadata-api"))

	rd.PipelineServiceAccount = corev1obj.NewServiceAccount(helper.SuffixObjectKey(key, "pipeline"))
	rd.UntrustedServiceAccount = corev1obj.NewServiceAccount(helper.SuffixObjectKey(key, "untrusted"))

	rd.MetadataAPIServiceAccountTokenSecrets = corev1obj.NewServiceAccountTokenSecrets(rd.MetadataAPIServiceAccount)

	ok, err := lifecycle.Loaders{
		rd.OwnerConfigMap,
		lifecycle.IgnoreNilLoader{Loader: rd.NetworkPolicy},
		rd.ImmutableConfigMap,
		rd.MutableConfigMap,
		rd.MetadataAPIServiceAccount,
		lifecycle.NewPrereqLoader(
			corev1obj.NewServiceAccountTokenSecretsDefaultPresentPoller(rd.MetadataAPIServiceAccountTokenSecrets),
			rd.MetadataAPIServiceAccount.Object,
		),
		rd.MetadataAPIRole,
		rd.MetadataAPIRoleBinding,
		rd.PipelineServiceAccount,
		rd.UntrustedServiceAccount,
	}.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	return &RunDepsLoadResult{
		Upstream: true,
		All:      ok,
	}, nil
}

func (rd *RunDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := rd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		rd.ImmutableConfigMap,
		rd.MutableConfigMap,
		rd.MetadataAPIServiceAccount,
		rd.MetadataAPIRole,
		rd.MetadataAPIRoleBinding,
		rd.PipelineServiceAccount,
		rd.UntrustedServiceAccount,
	}
	for _, o := range os {
		if err := rd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	if rd.NetworkPolicy != nil {
		if err := rd.OwnerConfigMap.Own(ctx, rd.NetworkPolicy); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		lifecycle.IgnoreNilPersister{Persister: rd.NetworkPolicy},
		rd.ImmutableConfigMap,
		rd.MutableConfigMap,
		rd.MetadataAPIServiceAccount,
		rd.MetadataAPIRole,
		rd.MetadataAPIRoleBinding,
		rd.PipelineServiceAccount,
		rd.UntrustedServiceAccount,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	// Sync token secrets.
	if _, err := corev1obj.NewServiceAccountTokenSecretsDefaultPresentPoller(rd.MetadataAPIServiceAccountTokenSecrets).Load(ctx, cl); err != nil {
		return err
	}

	return nil
}

func (rd *RunDeps) AnnotateStepToken(ctx context.Context, target *metav1.ObjectMeta, ws *relayv1beta1.Step) error {
	if _, found := target.Annotations[authenticate.KubernetesTokenAnnotation]; found {
		// We only add this once and exactly once per run per target.
		return nil
	}

	ms := ModelStep(rd.Run, ws)
	now := time.Now()

	// FIXME Temporarily avoid unknown transient issue
	if rd.MetadataAPIServiceAccountTokenSecrets == nil ||
		rd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret == nil ||
		rd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Object == nil {
		return errors.New("no default token secret set for run")
	}

	sat, err := rd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Token()
	if err != nil {
		return errmark.MarkTransientIf(err, errmark.RuleIs(corev1obj.ErrServiceAccountTokenMissingData))
	}

	annotations := rd.Run.Object.GetAnnotations()

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Issuer:    authenticate.ControllerIssuer,
			Audience:  jwt.Audience{authenticate.MetadataAPIAudienceV1},
			Subject:   path.Join(ms.Type().Plural, ms.Hash().HexEncoding()),
			Expiry:    jwt.NewNumericDate(now.Add(1*time.Hour + 5*time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},

		KubernetesNamespaceName:       rd.WorkflowDeps.TenantDeps.Namespace.Name,
		KubernetesNamespaceUID:        string(rd.WorkflowDeps.TenantDeps.Namespace.Object.GetUID()),
		KubernetesServiceAccountToken: sat,

		RelayDomainID: annotations[model.RelayDomainIDAnnotation],
		RelayTenantID: annotations[model.RelayTenantIDAnnotation],
		RelayRunID:    ms.Run.ID,
		RelayName:     ms.Name,

		RelayKubernetesImmutableConfigMapName: rd.ImmutableConfigMap.Key.Name,
		RelayKubernetesMutableConfigMapName:   rd.MutableConfigMap.Key.Name,

		RelayVaultEnginePath:     annotations[model.RelayVaultEngineMountAnnotation],
		RelayVaultSecretPath:     annotations[model.RelayVaultSecretPathAnnotation],
		RelayVaultConnectionPath: annotations[model.RelayVaultConnectionPathAnnotation],
	}

	td := rd.WorkflowDeps.TenantDeps
	if sink := td.APIWorkflowExecutionSink; sink != nil {
		if u, _ := url.Parse(sink.URL()); u != nil {
			claims.RelayWorkflowExecutionAPIURL = &types.URL{URL: u}
			claims.RelayWorkflowExecutionAPIToken, _ = sink.Token()
		}
	}

	tok, err := rd.Issuer.Issue(ctx, claims)
	if err != nil {
		return err
	}

	helper.Annotate(target, authenticate.KubernetesTokenAnnotation, string(tok))
	helper.Annotate(target, authenticate.KubernetesSubjectAnnotation, claims.Subject)

	return nil
}

type RunDepsOption func(rd *RunDeps)

func RunDepsWithStandaloneMode(standalone bool) RunDepsOption {
	return func(rd *RunDeps) {
		if standalone {
			rd.Standalone = true
		}
	}
}

func NewRunDeps(r *obj.Run, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...RunDepsOption) *RunDeps {
	rd := &RunDeps{
		Run: r,
		Workflow: obj.NewWorkflow(client.ObjectKey{
			Namespace: r.Key.Namespace,
			Name:      r.Object.Spec.WorkflowRef.Name,
		}),

		Issuer: issuer,

		MetadataAPIURL: metadataAPIURL,
	}

	for _, opt := range opts {
		opt(rd)
	}

	return rd
}

func ConfigureRunDeps(ctx context.Context, rd *RunDeps) error {
	if err := DependencyManager.SetDependencyOf(
		rd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: rd.Run.Object,
			GVK:    relayv1beta1.RunKind,
		}); err != nil {
		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		rd.ImmutableConfigMap,
		rd.MutableConfigMap,
		rd.MetadataAPIServiceAccount,
		rd.MetadataAPIRole,
		rd.PipelineServiceAccount,
		rd.UntrustedServiceAccount,
	}
	for _, laf := range lafs {
		laf.LabelAnnotateFrom(ctx, rd.Run.Object)
		lifecycle.Label(ctx, laf, model.RelayControllerWorkflowRunIDLabel, rd.Run.Key.Name)
	}

	if rd.Standalone {
		rd.NetworkPolicy.AllowAll()
	} else {
		ConfigureNetworkPolicyForRun(rd.NetworkPolicy, rd.Run)
	}

	if err := ConfigureImmutableConfigMapForRun(ctx, rd.ImmutableConfigMap, rd); err != nil {
		return err
	}
	if err := ConfigureMutableConfigMapForRun(ctx, rd.MutableConfigMap, rd.Run); err != nil {
		return err
	}

	ConfigureMetadataAPIServiceAccount(rd.MetadataAPIServiceAccount)
	ConfigureMetadataAPIRole(rd.MetadataAPIRole, rd.ImmutableConfigMap, rd.MutableConfigMap)
	ConfigureMetadataAPIRoleBinding(rd.MetadataAPIRoleBinding, rd.MetadataAPIServiceAccount, rd.MetadataAPIRole)
	ConfigureUntrustedServiceAccount(rd.PipelineServiceAccount)
	ConfigureUntrustedServiceAccount(rd.UntrustedServiceAccount)

	return nil
}

func ApplyRunDeps(ctx context.Context, cl client.Client, r *obj.Run, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...RunDepsOption) (*RunDeps, error) {
	rd := NewRunDeps(r, issuer, metadataAPIURL, opts...)

	if lr, err := rd.Load(ctx, cl); err != nil {
		return nil, err
	} else if !lr.Upstream {
		return nil, fmt.Errorf("waiting on Run upstream dependencies")
	}

	if err := ConfigureRunDeps(ctx, rd); err != nil {
		return nil, err
	}

	if err := rd.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return rd, nil
}
