package obj

import (
	"context"
	"crypto/sha256"
	"math"
	"net/url"
	"path"
	"time"

	"github.com/puppetlabs/horsehead/v2/jsonutil"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/hashutil"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// Go dates start at year 1, UNIX dates start at year 1970, so this is the
	// difference between 1970 and 1 in seconds. This constant exists as
	// `unixToInternal` in the Go standard library time package.
	maxUsableTime = time.Unix(math.MaxInt64-62135596800, 999999999)
)

type WebhookTriggerDepsLoadResult struct {
	Upstream bool
	All      bool
}

type WebhookTriggerDeps struct {
	WebhookTrigger *WebhookTrigger
	Issuer         authenticate.Issuer
	Tenant         *Tenant
	TenantDeps     *TenantDeps

	// StaleOwnerConfigMap is a reference to a now-outdated stub object that
	// needs to be cleaned up. It is set if the tenant is deleted or if the
	// tenant namespace changes.
	StaleOwnerConfigMap *ConfigMap

	// OwnerConfigMap is a stub object that allows us to aggregate ownership of
	// the other objects created by these dependencies in the tenant namespace.
	OwnerConfigMap *ConfigMap

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
var _ Deleter = &WebhookTriggerDeps{}

func (wtd *WebhookTriggerDeps) Persist(ctx context.Context, cl client.Client) error {
	if _, err := wtd.DeleteStale(ctx, cl); err != nil {
		return err
	}

	// Must have the owner UID before assigning ownership.
	if err := wtd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []Ownable{
		wtd.NetworkPolicy,
		wtd.ImmutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.MetadataAPIRoleBinding,
		wtd.KnativeServiceAccount,
	}
	for _, o := range os {
		if err := wtd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []Persister{
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

func (wtd *WebhookTriggerDeps) Load(ctx context.Context, cl client.Client) (*WebhookTriggerDepsLoadResult, error) {
	// Load tenant and tenant dependencies first so that we can resolve
	// everything else.
	if ok, err := wtd.Tenant.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		// In this case, our tenant may have been deleted so we check to see if
		// this trigger already has resources created. If so, we add the stale
		// config map at the current version.
		if wtd.WebhookTrigger.Object.Status.Namespace != "" {
			wtd.StaleOwnerConfigMap = NewConfigMap(SuffixObjectKey(client.ObjectKey{
				Namespace: wtd.WebhookTrigger.Object.Status.Namespace,
				Name:      wtd.WebhookTrigger.Key.Name,
			}, "owner"))

			if _, err := wtd.StaleOwnerConfigMap.Load(ctx, cl); err != nil {
				return nil, err
			}
		}

		return &WebhookTriggerDepsLoadResult{}, nil
	}

	wtd.TenantDeps = NewTenantDeps(wtd.Tenant)

	if ok, err := wtd.TenantDeps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		// Waiting for tenant to settle now.
		return &WebhookTriggerDepsLoadResult{}, nil
	}

	// Key will be our webhook trigger name *in* the tenant namespace.
	key := client.ObjectKey{Namespace: wtd.TenantDeps.Namespace.Name, Name: wtd.WebhookTrigger.Key.Name}

	wtd.OwnerConfigMap = NewConfigMap(SuffixObjectKey(key, "owner"))
	if wtd.WebhookTrigger.Object.Status.Namespace != "" && wtd.OwnerConfigMap.Key.Namespace != wtd.WebhookTrigger.Object.Status.Namespace {
		// In this case, the configuration of the tenant has changed. We'll
		// delete the current owner map and replace it with the new one.
		wtd.StaleOwnerConfigMap = NewConfigMap(SuffixObjectKey(client.ObjectKey{
			Namespace: wtd.WebhookTrigger.Object.Status.Namespace,
			Name:      wtd.WebhookTrigger.Key.Name,
		}, "owner"))
	}

	wtd.NetworkPolicy = NewNetworkPolicy(key)

	wtd.ImmutableConfigMap = NewConfigMap(SuffixObjectKey(key, "immutable"))
	wtd.MutableConfigMap = NewConfigMap(SuffixObjectKey(key, "mutable"))

	wtd.MetadataAPIServiceAccount = NewServiceAccount(SuffixObjectKey(key, "metadata-api"))
	wtd.MetadataAPIRole = NewRole(SuffixObjectKey(key, "metadata-api"))
	wtd.MetadataAPIRoleBinding = NewRoleBinding(SuffixObjectKey(key, "metadata-api"))

	wtd.KnativeServiceAccount = NewServiceAccount(SuffixObjectKey(key, "knative"))

	ok, err := Loaders{
		IgnoreNilLoader{wtd.StaleOwnerConfigMap},
		wtd.OwnerConfigMap,
		wtd.NetworkPolicy,
		wtd.ImmutableConfigMap,
		wtd.MutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.MetadataAPIRoleBinding,
		wtd.KnativeServiceAccount,
	}.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	return &WebhookTriggerDepsLoadResult{
		Upstream: true,
		All:      ok,
	}, nil
}

func (wtd *WebhookTriggerDeps) Delete(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := wtd.DeleteStale(ctx, cl); err != nil {
		return false, err
	}

	if wtd.OwnerConfigMap == nil || wtd.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := IsDependencyOf(wtd.OwnerConfigMap.Object.ObjectMeta, Owner{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return false, err
	} else if ok {
		return wtd.OwnerConfigMap.Delete(ctx, cl)
	}

	return true, nil
}

func (wtd *WebhookTriggerDeps) DeleteStale(ctx context.Context, cl client.Client) (bool, error) {
	if wtd.StaleOwnerConfigMap == nil || wtd.StaleOwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := IsDependencyOf(wtd.StaleOwnerConfigMap.Object.ObjectMeta, Owner{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return false, err
	} else if ok {
		return wtd.StaleOwnerConfigMap.Delete(ctx, cl)
	}

	return true, nil
}

func (wtd *WebhookTriggerDeps) AnnotateTriggerToken(ctx context.Context, target *metav1.ObjectMeta) error {
	idh := hashutil.NewStructuredHash(sha256.New)

	mt := ModelWebhookTrigger(wtd.WebhookTrigger)
	now := time.Now()

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Issuer:   authenticate.ControllerIssuer,
			Audience: jwt.Audience{authenticate.MetadataAPIAudienceV1},
			// When used with Vault, if no expiry is specified, their
			// authenticator will default to a 5 minute expiration. We want
			// these tokens to be indefinite (we'll rotate the issuer if
			// anything bad happens since it's only used internally), so we set
			// the expiration to the maximum possible value.
			Expiry:    jwt.NewNumericDate(maxUsableTime),
			Subject:   path.Join(mt.Type().Plural, mt.Hash().HexEncoding()),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	idh.Set("subject", claims.Subject)

	sat, err := wtd.MetadataAPIServiceAccount.DefaultTokenSecret.Token()
	if err != nil {
		return err
	}

	annotations := wtd.WebhookTrigger.Object.GetAnnotations()

	claims.KubernetesNamespaceName = wtd.TenantDeps.Namespace.Name
	claims.KubernetesNamespaceUID = string(wtd.TenantDeps.Namespace.Object.GetUID())
	claims.KubernetesServiceAccountToken = sat

	claims.RelayDomainID = annotations[model.RelayDomainIDAnnotation]
	claims.RelayTenantID = annotations[model.RelayTenantIDAnnotation]
	claims.RelayName = mt.Name
	idh.Set("parents", claims.RelayDomainID, claims.RelayTenantID)

	claims.RelayKubernetesImmutableConfigMapName = wtd.ImmutableConfigMap.Key.Name
	claims.RelayKubernetesMutableConfigMapName = wtd.MutableConfigMap.Key.Name

	claims.RelayVaultEnginePath = annotations[model.RelayVaultEngineMountAnnotation]
	claims.RelayVaultSecretPath = annotations[model.RelayVaultSecretPathAnnotation]
	claims.RelayVaultConnectionPath = annotations[model.RelayVaultConnectionPathAnnotation]
	idh.Set("vault", claims.RelayVaultEnginePath, claims.RelayVaultSecretPath, claims.RelayVaultConnectionPath)

	if sink := wtd.TenantDeps.APITriggerEventSink; sink != nil {
		if u, _ := url.Parse(sink.URL()); u != nil {
			claims.RelayEventAPIURL = &jsonutil.URL{URL: u}
			claims.RelayEventAPIToken, _ = sink.Token()
			idh.Set("event", claims.RelayEventAPIURL.String(), claims.RelayEventAPIToken)
		}
	}

	if h, err := idh.Sum(); err != nil {
		return err
	} else if enc := h.HexEncoding(); enc != target.GetAnnotations()[model.RelayControllerTokenHashAnnotation] {
		tok, err := wtd.Issuer.Issue(ctx, claims)
		if err != nil {
			return err
		}

		Annotate(target, model.RelayControllerTokenHashAnnotation, enc)
		Annotate(target, authenticate.KubernetesTokenAnnotation, string(tok))
		Annotate(target, authenticate.KubernetesSubjectAnnotation, claims.Subject)
	}

	return nil
}

func NewWebhookTriggerDeps(wt *WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL) *WebhookTriggerDeps {
	key := wt.Key

	return &WebhookTriggerDeps{
		WebhookTrigger: wt,
		Issuer:         issuer,

		Tenant: NewTenant(client.ObjectKey{Namespace: key.Namespace, Name: wt.Object.Spec.TenantRef.Name}),

		MetadataAPIURL: metadataAPIURL,
	}
}

func ConfigureWebhookTriggerDeps(ctx context.Context, wtd *WebhookTriggerDeps) error {
	// Set up the owner config map as the target for the finalizer.
	SetDependencyOf(&wtd.OwnerConfigMap.Object.ObjectMeta, Owner{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind})

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

func ApplyWebhookTriggerDeps(ctx context.Context, cl client.Client, wt *WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL) (*WebhookTriggerDeps, error) {
	deps := NewWebhookTriggerDeps(wt, issuer, metadataAPIURL)

	if loaded, err := deps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !loaded.Upstream {
		return nil, ErrRequired
	}

	if err := ConfigureWebhookTriggerDeps(ctx, deps); err != nil {
		return nil, err
	}

	if err := deps.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return deps, nil
}
