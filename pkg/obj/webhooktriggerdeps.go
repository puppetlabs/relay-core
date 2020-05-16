package obj

import (
	"context"
	"net/url"
	"path"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"gopkg.in/square/go-jose.v2/jwt"
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

	KnativeServiceAccount *ServiceAccount
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
		wtd.KnativeServiceAccount,
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
		wtd.KnativeServiceAccount,
	}.Load(ctx, cl)
}

func (wtd *WebhookTriggerDeps) AnnotateTriggerToken(ctx context.Context, target *metav1.ObjectMeta) error {
	if tok := target.Annotations[authenticate.KubernetesTokenAnnotation]; tok != "" {
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
			Issuer:   authenticate.ControllerIssuer,
			Audience: jwt.Audience{authenticate.MetadataAPIAudienceV1},
			Subject:  path.Join(mt.Type().Plural, mt.Hash().HexEncoding()),
			// TODO: Do we want any expiry on these?
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

func NewWebhookTriggerDeps(wt *WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL) *WebhookTriggerDeps {
	key := wt.Key

	return &WebhookTriggerDeps{
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

		KnativeServiceAccount: NewServiceAccount(SuffixObjectKey(key, "knative")),
	}
}

func ConfigureTriggerDeps(ctx context.Context, wtd *WebhookTriggerDeps) error {
	os := []Ownable{
		wtd.LimitRange,
		wtd.NetworkPolicy,
		wtd.ImmutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.MetadataAPIRoleBinding,
		wtd.KnativeServiceAccount,
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
		wtd.KnativeServiceAccount,
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

	ConfigureUntrustedServiceAccount(wtd.KnativeServiceAccount)

	return nil
}

func ApplyTriggerDeps(ctx context.Context, cl client.Client, wt *WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL) (*WebhookTriggerDeps, error) {
	deps := NewWebhookTriggerDeps(wt, issuer, metadataAPIURL)

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
