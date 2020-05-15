package obj

import (
	"context"
	"net/url"
	"path"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"gopkg.in/square/go-jose.v2/jwt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WebhookTriggerDeps struct {
	WebhookTrigger *WebhookTrigger
	Issuer         authenticate.Issuer

	Namespace *Namespace

	// TODO: This belongs at the Tenant as it should apply to the whole
	// namespace.
	LimitRange *LimitRange

	NetworkPolicy *NetworkPolicy

	ImmutableConfigMap *ConfigMap
	MutableConfigMap   *ConfigMap

	MetadataAPIURL            *url.URL
	MetadataAPIServiceAccount *ServiceAccount
	MetadataAPIRole           *Role
	MetadataAPIRoleBinding    *RoleBinding

	SourceSystemImagePullSecret *ImagePullSecret
	TargetSystemImagePullSecret *ImagePullSecret
	SystemServiceAccount        *ServiceAccount
}

var _ Persister = &WebhookTriggerDeps{}
var _ Loader = &WebhookTriggerDeps{}

func (wtd *WebhookTriggerDeps) Persist(ctx context.Context, cl client.Client) error {
	ps := []Persister{
		wtd.LimitRange,
		wtd.NetworkPolicy,
		wtd.ImmutableConfigMap,
		wtd.MutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.MetadataAPIRoleBinding,
		IgnoreNilPersister{wtd.TargetSystemImagePullSecret},
		wtd.SystemServiceAccount,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (wtd *WebhookTriggerDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	return Loaders{
		RequiredLoader{wtd.Namespace},
		wtd.LimitRange,
		wtd.NetworkPolicy,
		wtd.ImmutableConfigMap,
		wtd.MutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.MetadataAPIRoleBinding,
		RequiredLoader{IgnoreNilLoader{wtd.SourceSystemImagePullSecret}},
		IgnoreNilLoader{wtd.TargetSystemImagePullSecret},
		wtd.SystemServiceAccount,
	}.Load(ctx, cl)
}

func (wtd *WebhookTriggerDeps) AnnotateTriggerToken(ctx context.Context, target *metav1.ObjectMeta) error {
	if _, found := target.Annotations[authenticate.KubernetesTokenAnnotation]; found {
		// We only add this once and exactly once per run per target.
		return nil
	}

	mt := ModelTrigger(wtd.WebhookTrigger)
	now := time.Now()

	sat, err := wtd.MetadataAPIServiceAccount.DefaultTokenSecret.Token()
	if err != nil {
		return err
	}

	annotations := wtd.WebhookTrigger.Object.GetAnnotations()

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Issuer:    authenticate.ControllerIssuer,
			Audience:  jwt.Audience{authenticate.MetadataAPIAudienceV1},
			Subject:   path.Join(mt.Type().Plural, mt.Hash().HexEncoding()),
			Expiry:    jwt.NewNumericDate(now.Add(1*time.Hour + 5*time.Minute)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},

		KubernetesNamespaceName:       wtd.Namespace.Name,
		KubernetesNamespaceUID:        string(wtd.Namespace.Object.GetUID()),
		KubernetesServiceAccountToken: sat,

		RelayDomainID: annotations[model.RelayDomainIDAnnotation],
		RelayTenantID: annotations[model.RelayTenantIDAnnotation],
		RelayName:     mt.Name,

		RelayKubernetesImmutableConfigMapName: wtd.ImmutableConfigMap.Key.Name,
		RelayKubernetesMutableConfigMapName:   wtd.MutableConfigMap.Key.Name,

		RelayVaultEnginePath:     annotations[model.RelayVaultEngineMountAnnotation],
		RelayVaultSecretPath:     annotations[model.RelayVaultSecretPathAnnotation],
		RelayVaultConnectionPath: annotations[model.RelayVaultConnectionPathAnnotation],
	}

	tok, err := wtd.Issuer.Issue(ctx, claims)
	if err != nil {
		return err
	}

	Annotate(target, authenticate.KubernetesTokenAnnotation, string(tok))
	Annotate(target, authenticate.KubernetesSubjectAnnotation, claims.Subject)

	return nil
}

type WebhookTriggerDepsOption func(wtd *WebhookTriggerDeps)

func WebhookTriggerDepsWithSourceSystemImagePullSecret(key client.ObjectKey) WebhookTriggerDepsOption {
	return func(wtd *WebhookTriggerDeps) {
		wtd.SourceSystemImagePullSecret = NewImagePullSecret(key)
		wtd.TargetSystemImagePullSecret = NewImagePullSecret(SuffixObjectKey(wtd.WebhookTrigger.Key, "system"))
	}
}

func NewWebhookTriggerDeps(wt *WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WebhookTriggerDepsOption) *WebhookTriggerDeps {
	key := wt.Key

	pd := &WebhookTriggerDeps{
		WebhookTrigger: wt,
		Issuer:         issuer,

		Namespace: NewNamespace(key.Namespace),

		LimitRange: NewLimitRange(key),

		NetworkPolicy: NewNetworkPolicy(key),

		ImmutableConfigMap: NewConfigMap(SuffixObjectKey(key, "immutable")),
		MutableConfigMap:   NewConfigMap(SuffixObjectKey(key, "mutable")),

		MetadataAPIURL:            metadataAPIURL,
		MetadataAPIServiceAccount: NewServiceAccount(SuffixObjectKey(key, "metadata-api")),
		MetadataAPIRole:           NewRole(SuffixObjectKey(key, "metadata-api")),
		MetadataAPIRoleBinding:    NewRoleBinding(SuffixObjectKey(key, "metadata-api")),

		SystemServiceAccount: NewServiceAccount(SuffixObjectKey(key, "system")),
	}

	for _, opt := range opts {
		opt(pd)
	}

	return pd
}

func ConfigureTriggerDeps(ctx context.Context, wtd *WebhookTriggerDeps) error {
	os := []Ownable{
		wtd.LimitRange,
		wtd.NetworkPolicy,
		wtd.ImmutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.MetadataAPIRoleBinding,
		IgnoreNilOwnable{wtd.TargetSystemImagePullSecret},
		wtd.SystemServiceAccount,
	}
	for _, o := range os {
		if err := wtd.WebhookTrigger.Own(ctx, o); err != nil {
			return err
		}
	}

	lafs := []LabelAnnotatableFrom{
		wtd.ImmutableConfigMap,
		wtd.MutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.SystemServiceAccount,
	}
	for _, laf := range lafs {
		laf.LabelAnnotateFrom(ctx, wtd.WebhookTrigger.Object.ObjectMeta)
	}

	ConfigureLimitRange(wtd.LimitRange)

	ConfigureNetworkPolicyForWebhookTrigger(wtd.NetworkPolicy, wtd.WebhookTrigger)

	if err := ConfigureImmutableConfigMapForWebhookTrigger(ctx, wtd.ImmutableConfigMap, wtd.WebhookTrigger); err != nil {
		return err
	}

	ConfigureMetadataAPIServiceAccount(wtd.MetadataAPIServiceAccount)
	ConfigureMetadataAPIRole(wtd.MetadataAPIRole, wtd.ImmutableConfigMap, wtd.MutableConfigMap)
	ConfigureMetadataAPIRoleBinding(wtd.MetadataAPIRoleBinding, wtd.MetadataAPIServiceAccount, wtd.MetadataAPIRole)

	{
		var opts []SystemServiceAccountOption
		if wtd.SourceSystemImagePullSecret != nil {
			ConfigureImagePullSecret(wtd.TargetSystemImagePullSecret, wtd.SourceSystemImagePullSecret)
			opts = append(opts, SystemServiceAccountWithImagePullSecret(corev1.LocalObjectReference{Name: wtd.TargetSystemImagePullSecret.Key.Name}))
		}

		ConfigureSystemServiceAccount(wtd.SystemServiceAccount, opts...)
	}

	return nil
}

func ApplyTriggerDeps(ctx context.Context, cl client.Client, wt *WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WebhookTriggerDepsOption) (*WebhookTriggerDeps, error) {
	deps := NewWebhookTriggerDeps(wt, issuer, metadataAPIURL, opts...)

	if _, err := deps.Load(ctx, cl); err != nil {
		return nil, err
	}

	if err := ConfigureTriggerDeps(ctx, deps); err != nil {
		return nil, err
	}

	if err := deps.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return deps, nil
}
