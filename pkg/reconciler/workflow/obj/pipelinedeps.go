package obj

import (
	"context"
	"net/url"
	"path"
	"time"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"gopkg.in/square/go-jose.v2/jwt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PipelineDeps represents the Kubernetes objects required to create a Pipeline.
type PipelineDeps struct {
	WorkflowRun *WorkflowRun
	Issuer      authenticate.Issuer

	Namespace *Namespace

	// TODO: These belong at the Tenant as they should apply to the whole
	// namespace.
	NetworkPolicy *NetworkPolicy
	LimitRange    *LimitRange

	ImmutableConfigMap *ConfigMap
	MutableConfigMap   *ConfigMap

	MetadataAPIURL            *url.URL
	MetadataAPIServiceAccount *ServiceAccount
	MetadataAPIRole           *Role
	MetadataAPIRoleBinding    *RoleBinding

	PipelineServiceAccount *ServiceAccount

	SourceSystemImagePullSecret *ImagePullSecret
	TargetSystemImagePullSecret *ImagePullSecret
	SystemServiceAccount        *ServiceAccount
}

var _ Persister = &PipelineDeps{}
var _ Loader = &PipelineDeps{}

func (pd *PipelineDeps) Persist(ctx context.Context, cl client.Client) error {
	ps := []Persister{
		pd.NetworkPolicy,
		pd.LimitRange,
		pd.ImmutableConfigMap,
		pd.MutableConfigMap,
		pd.MetadataAPIServiceAccount,
		pd.MetadataAPIRole,
		pd.MetadataAPIRoleBinding,
		pd.PipelineServiceAccount,
		IgnoreNilPersister{pd.TargetSystemImagePullSecret},
		pd.SystemServiceAccount,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (pd *PipelineDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	return Loaders{
		RequiredLoader{pd.Namespace},
		pd.NetworkPolicy,
		pd.LimitRange,
		pd.ImmutableConfigMap,
		pd.MutableConfigMap,
		pd.MetadataAPIServiceAccount,
		pd.MetadataAPIRole,
		pd.MetadataAPIRoleBinding,
		pd.PipelineServiceAccount,
		RequiredLoader{IgnoreNilLoader{pd.SourceSystemImagePullSecret}},
		IgnoreNilLoader{pd.TargetSystemImagePullSecret},
		pd.SystemServiceAccount,
	}.Load(ctx, cl)
}

func (pd *PipelineDeps) AnnotateStepToken(ctx context.Context, target *metav1.ObjectMeta, ws *nebulav1.WorkflowStep) error {
	if _, found := target.Annotations[authenticate.KubernetesTokenAnnotation]; found {
		// We only add this once and exactly once per run per target.
		return nil
	}

	ms := ModelStep(pd.WorkflowRun, ws)
	now := time.Now()

	sat, err := pd.MetadataAPIServiceAccount.DefaultTokenSecret.Token()
	if err != nil {
		return err
	}

	annotations := pd.WorkflowRun.Object.GetAnnotations()

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Issuer:    authenticate.ControllerIssuer,
			Audience:  jwt.Audience{authenticate.MetadataAPIAudienceV1},
			Subject:   path.Join(ms.Type().Plural, ms.Hash().HexEncoding()),
			Expiry:    jwt.NewNumericDate(now.Add(1*time.Hour + 5*time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},

		KubernetesNamespaceName:       pd.Namespace.Name,
		KubernetesNamespaceUID:        string(pd.Namespace.Object.GetUID()),
		KubernetesServiceAccountToken: sat,

		RelayDomainID: annotations[WorkflowRunDomainIDAnnotation],
		RelayTenantID: annotations[WorkflowRunTenantIDAnnotation],
		RelayRunID:    ms.Run.ID,
		RelayName:     ms.Name,

		RelayKubernetesImmutableConfigMapName: pd.ImmutableConfigMap.Key.Name,
		RelayKubernetesMutableConfigMapName:   pd.MutableConfigMap.Key.Name,

		RelayVaultSecretPath: annotations[WorkflowRunVaultSecretPathAnnotation],
	}

	tok, err := pd.Issuer.Issue(ctx, claims)
	if err != nil {
		return err
	}

	Annotate(target, authenticate.KubernetesTokenAnnotation, string(tok))
	Annotate(target, authenticate.KubernetesSubjectAnnotation, claims.Subject)

	return nil
}

type PipelineDepsOption func(pd *PipelineDeps)

func PipelineDepsWithSourceSystemImagePullSecret(key client.ObjectKey) PipelineDepsOption {
	return func(pd *PipelineDeps) {
		pd.SourceSystemImagePullSecret = NewImagePullSecret(key)
		pd.TargetSystemImagePullSecret = NewImagePullSecret(SuffixObjectKey(pd.WorkflowRun.Key, "system"))
	}
}

func NewPipelineDeps(wr *WorkflowRun, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...PipelineDepsOption) *PipelineDeps {
	key := wr.Key

	pd := &PipelineDeps{
		WorkflowRun: wr,
		Issuer:      issuer,

		Namespace: NewNamespace(key.Namespace),

		NetworkPolicy: NewNetworkPolicy(key),
		LimitRange:    NewLimitRange(key),

		ImmutableConfigMap: NewConfigMap(SuffixObjectKey(key, "immutable")),
		MutableConfigMap:   NewConfigMap(SuffixObjectKey(key, "mutable")),

		MetadataAPIURL:            metadataAPIURL,
		MetadataAPIServiceAccount: NewServiceAccount(SuffixObjectKey(key, "metadata-api")),
		MetadataAPIRole:           NewRole(SuffixObjectKey(key, "metadata-api")),
		MetadataAPIRoleBinding:    NewRoleBinding(SuffixObjectKey(key, "metadata-api")),

		PipelineServiceAccount: NewServiceAccount(SuffixObjectKey(key, "pipeline")),

		SystemServiceAccount: NewServiceAccount(SuffixObjectKey(key, "system")),
	}

	for _, opt := range opts {
		opt(pd)
	}

	return pd
}

func ConfigurePipelineDeps(ctx context.Context, pd *PipelineDeps) error {
	os := []Ownable{
		pd.NetworkPolicy,
		pd.LimitRange,
		pd.ImmutableConfigMap,
		pd.MutableConfigMap,
		pd.MetadataAPIServiceAccount,
		pd.MetadataAPIRole,
		pd.MetadataAPIRoleBinding,
		pd.PipelineServiceAccount,
		IgnoreNilOwnable{pd.TargetSystemImagePullSecret},
		pd.SystemServiceAccount,
	}
	for _, o := range os {
		pd.WorkflowRun.Own(ctx, o)
	}

	lafs := []LabelAnnotatableFrom{
		pd.ImmutableConfigMap,
		pd.MutableConfigMap,
		pd.MetadataAPIServiceAccount,
		pd.MetadataAPIRole,
		pd.PipelineServiceAccount,
		pd.SystemServiceAccount,
	}
	for _, laf := range lafs {
		laf.LabelAnnotateFrom(ctx, pd.WorkflowRun.Object.ObjectMeta)
	}

	ConfigureNetworkPolicy(pd.NetworkPolicy)
	ConfigureLimitRange(pd.LimitRange)

	if err := ConfigureImmutableConfigMap(ctx, pd.ImmutableConfigMap, pd.WorkflowRun); err != nil {
		return err
	}
	if err := ConfigureMutableConfigMap(ctx, pd.MutableConfigMap, pd.WorkflowRun); err != nil {
		return err
	}

	ConfigureMetadataAPIServiceAccount(pd.MetadataAPIServiceAccount)
	ConfigureMetadataAPIRole(pd.MetadataAPIRole, pd.ImmutableConfigMap, pd.MutableConfigMap)
	ConfigureMetadataAPIRoleBinding(pd.MetadataAPIRoleBinding, pd.MetadataAPIServiceAccount, pd.MetadataAPIRole)

	ConfigurePipelineServiceAccount(pd.PipelineServiceAccount)

	{
		var opts []SystemServiceAccountOption
		if pd.SourceSystemImagePullSecret != nil {
			ConfigureImagePullSecret(pd.TargetSystemImagePullSecret, pd.SourceSystemImagePullSecret)
			opts = append(opts, SystemServiceAccountWithImagePullSecret(corev1.LocalObjectReference{Name: pd.TargetSystemImagePullSecret.Key.Name}))
		}

		ConfigureSystemServiceAccount(pd.SystemServiceAccount, opts...)
	}

	return nil
}

func ApplyPipelineDeps(ctx context.Context, cl client.Client, wr *WorkflowRun, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...PipelineDepsOption) (*PipelineDeps, error) {
	deps := NewPipelineDeps(wr, issuer, metadataAPIURL, opts...)

	if _, err := deps.Load(ctx, cl); err != nil {
		return nil, err
	}

	if err := ConfigurePipelineDeps(ctx, deps); err != nil {
		return nil, err
	}

	if err := deps.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return deps, nil
}
