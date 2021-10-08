package app

import (
	"context"
	"crypto/sha256"
	"math"
	"net/url"
	"path"
	"time"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/hashutil"
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
	WebhookTrigger *obj.WebhookTrigger
	Issuer         authenticate.Issuer
	Tenant         *obj.Tenant
	TenantDeps     *TenantDeps

	ToolInjectionPoolRef pvpoolv1alpha1.PoolReference
	Standalone           bool

	// StaleOwnerConfigMap is a reference to a now-outdated stub object that
	// needs to be cleaned up. It is set if the tenant is deleted or if the
	// tenant namespace changes.
	StaleOwnerConfigMap *corev1obj.ConfigMap

	// OwnerConfigMap is a stub object that allows us to aggregate ownership of
	// the other objects created by these dependencies in the tenant namespace.
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

	KnativeServiceAccount *corev1obj.ServiceAccount
}

var _ lifecycle.Deleter = &WebhookTriggerDeps{}
var _ lifecycle.Persister = &WebhookTriggerDeps{}

func (wtd *WebhookTriggerDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if _, err := wtd.DeleteStale(ctx, cl, opts...); err != nil {
		return false, err
	}

	if wtd.OwnerConfigMap == nil || wtd.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(wtd.OwnerConfigMap.Object, lifecycle.TypedObject{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return false, err
	} else if ok {
		return wtd.OwnerConfigMap.Delete(ctx, cl, opts...)
	}

	return true, nil
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
			wtd.StaleOwnerConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(client.ObjectKey{
				Namespace: wtd.WebhookTrigger.Object.Status.Namespace,
				Name:      wtd.WebhookTrigger.Key.Name,
			}, "owner"))

			if _, err := wtd.StaleOwnerConfigMap.Load(ctx, cl); err != nil {
				return nil, err
			}
		}

		return &WebhookTriggerDepsLoadResult{}, nil
	}

	wtd.TenantDeps = NewTenantDeps(wtd.Tenant, TenantDepsWithStandaloneMode(wtd.Standalone))

	if ok, err := wtd.TenantDeps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		// Waiting for tenant to settle now.
		return &WebhookTriggerDepsLoadResult{}, nil
	}

	// Key will be our webhook trigger name *in* the tenant namespace.
	key := client.ObjectKey{Namespace: wtd.TenantDeps.Namespace.Name, Name: wtd.WebhookTrigger.Key.Name}

	wtd.OwnerConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "owner"))
	if wtd.WebhookTrigger.Object.Status.Namespace != "" && wtd.OwnerConfigMap.Key.Namespace != wtd.WebhookTrigger.Object.Status.Namespace {
		// In this case, the configuration of the tenant has changed. We'll
		// delete the current owner map and replace it with the new one.
		wtd.StaleOwnerConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(client.ObjectKey{
			Namespace: wtd.WebhookTrigger.Object.Status.Namespace,
			Name:      wtd.WebhookTrigger.Key.Name,
		}, "owner"))
	}

	wtd.NetworkPolicy = networkingv1obj.NewNetworkPolicy(key)

	wtd.ToolInjectionCheckout = &PoolRefPredicatedCheckout{
		Checkout: pvpoolv1alpha1obj.NewCheckout(SuffixObjectKeyWithHashOfObjectKey(SuffixObjectKey(key, "tools"), client.ObjectKey{
			Namespace: wtd.ToolInjectionPoolRef.Namespace,
			Name:      wtd.ToolInjectionPoolRef.Name,
		})),
	}

	wtd.ImmutableConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "immutable"))
	wtd.MutableConfigMap = corev1obj.NewConfigMap(SuffixObjectKey(key, "mutable"))

	wtd.MetadataAPIServiceAccount = corev1obj.NewServiceAccount(SuffixObjectKey(key, "metadata-api"))
	wtd.MetadataAPIServiceAccountTokenSecrets = corev1obj.NewServiceAccountTokenSecrets(wtd.MetadataAPIServiceAccount)
	wtd.MetadataAPIRole = rbacv1obj.NewRole(SuffixObjectKey(key, "metadata-api"))
	wtd.MetadataAPIRoleBinding = rbacv1obj.NewRoleBinding(SuffixObjectKey(key, "metadata-api"))

	wtd.KnativeServiceAccount = corev1obj.NewServiceAccount(SuffixObjectKey(key, "knative"))

	ok, err := lifecycle.Loaders{
		lifecycle.IgnoreNilLoader{Loader: wtd.StaleOwnerConfigMap},
		wtd.OwnerConfigMap,
		wtd.NetworkPolicy,
		wtd.ToolInjectionCheckout,
		wtd.ImmutableConfigMap,
		wtd.MutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		lifecycle.NewPrereqLoader(
			corev1obj.NewServiceAccountTokenSecretsDefaultPresentPoller(wtd.MetadataAPIServiceAccountTokenSecrets),
			wtd.MetadataAPIServiceAccount.Object,
		),
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

func (wtd *WebhookTriggerDeps) Persist(ctx context.Context, cl client.Client) error {
	if _, err := wtd.DeleteStale(ctx, cl); err != nil {
		return err
	}

	// Must have the owner UID before assigning ownership.
	if err := wtd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		wtd.NetworkPolicy,
		wtd.ToolInjectionCheckout,
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

	ps := []lifecycle.Persister{
		wtd.NetworkPolicy,
		wtd.ToolInjectionCheckout,
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

	// Sync token secrets.
	if _, err := corev1obj.NewServiceAccountTokenSecretsDefaultPresentPoller(wtd.MetadataAPIServiceAccountTokenSecrets).Load(ctx, cl); err != nil {
		return err
	}

	return nil
}

func (wtd *WebhookTriggerDeps) DeleteStale(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if wtd.StaleOwnerConfigMap == nil || wtd.StaleOwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(wtd.StaleOwnerConfigMap.Object, lifecycle.TypedObject{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return false, err
	} else if ok {
		return wtd.StaleOwnerConfigMap.Delete(ctx, cl, opts...)
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

	sat, err := wtd.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Token()
	if err != nil {
		return errmark.MarkTransientIf(err, errmark.RuleIs(corev1obj.ErrServiceAccountTokenMissingData))
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
			claims.RelayEventAPIURL = &types.URL{URL: u}
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

		helper.Annotate(target, model.RelayControllerTokenHashAnnotation, enc)
		helper.Annotate(target, authenticate.KubernetesTokenAnnotation, string(tok))
		helper.Annotate(target, authenticate.KubernetesSubjectAnnotation, claims.Subject)
	}

	return nil
}

type WebhookTriggerDepsOption func(wtd *WebhookTriggerDeps)

func WebhookTriggerDepsWithStandaloneMode(standalone bool) WebhookTriggerDepsOption {
	return func(wtd *WebhookTriggerDeps) {
		wtd.Standalone = standalone
	}
}

func WebhookTriggerDepsWithToolInjectionPool(pr pvpoolv1alpha1.PoolReference) WebhookTriggerDepsOption {
	return func(wtd *WebhookTriggerDeps) {
		wtd.ToolInjectionPoolRef = pr
	}
}

func NewWebhookTriggerDeps(wt *obj.WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL, opts ...WebhookTriggerDepsOption) *WebhookTriggerDeps {
	key := wt.Key

	wtd := &WebhookTriggerDeps{
		WebhookTrigger: wt,
		Issuer:         issuer,

		Tenant: obj.NewTenant(client.ObjectKey{Namespace: key.Namespace, Name: wt.Object.Spec.TenantRef.Name}),

		MetadataAPIURL: metadataAPIURL,
	}

	for _, opt := range opts {
		opt(wtd)
	}

	return wtd
}

func ConfigureWebhookTriggerDeps(ctx context.Context, wtd *WebhookTriggerDeps) error {
	// Set up the owner config map as the target for the finalizer.
	if err := DependencyManager.SetDependencyOf(wtd.OwnerConfigMap.Object, lifecycle.TypedObject{Object: wtd.WebhookTrigger.Object, GVK: relayv1beta1.WebhookTriggerKind}); err != nil {
		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		wtd.ToolInjectionCheckout,
		wtd.ImmutableConfigMap,
		wtd.MutableConfigMap,
		wtd.MetadataAPIServiceAccount,
		wtd.MetadataAPIRole,
		wtd.KnativeServiceAccount,
	}
	for _, laf := range lafs {
		laf.LabelAnnotateFrom(ctx, wtd.WebhookTrigger.Object)
		lifecycle.Label(ctx, laf, model.RelayControllerWebhookTriggerIDLabel, wtd.WebhookTrigger.Key.Name)
	}

	if wtd.Standalone {
		wtd.NetworkPolicy.AllowAll()
	} else {
		ConfigureNetworkPolicyForWebhookTrigger(wtd.NetworkPolicy, wtd.WebhookTrigger)
	}

	ConfigureToolInjectionCheckout(wtd.ToolInjectionCheckout, wtd.TenantDeps.Tenant, wtd.ToolInjectionPoolRef)

	if err := ConfigureImmutableConfigMapForWebhookTrigger(ctx, wtd.ImmutableConfigMap, wtd.WebhookTrigger); err != nil {
		return err
	}

	ConfigureMetadataAPIServiceAccount(wtd.MetadataAPIServiceAccount)
	ConfigureMetadataAPIRole(wtd.MetadataAPIRole, wtd.ImmutableConfigMap, wtd.MutableConfigMap)
	ConfigureMetadataAPIRoleBinding(wtd.MetadataAPIRoleBinding, wtd.MetadataAPIServiceAccount, wtd.MetadataAPIRole)

	ConfigureUntrustedServiceAccount(wtd.KnativeServiceAccount)

	return nil
}
