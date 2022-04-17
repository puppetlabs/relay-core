package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/admissionregistrationv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	_ lifecycle.Loader    = &OperatorDeps{}
	_ lifecycle.Persister = &OperatorDeps{}
	_ obj.Configurable    = &OperatorDeps{}
)

type OperatorDeps struct {
	Core                             *obj.Core
	Deployment                       *operatorDeployment
	WebhookService                   *operatorWebhookService
	ServiceAccount                   *corev1obj.ServiceAccount
	TenantNamespace                  *corev1obj.Namespace
	ClusterRole                      *rbacv1obj.ClusterRole
	ClusterRoleBinding               *rbacv1obj.ClusterRoleBinding
	DelegateClusterRole              *rbacv1obj.ClusterRole
	DelegateClusterRoleBinding       *rbacv1obj.ClusterRoleBinding
	WebhookConfig                    *admissionregistrationv1.MutatingWebhookConfiguration
	OwnerConfigMap                   *corev1obj.ConfigMap
	WebhookCertificateControllerDeps *WebhookCertificateControllerDeps
	VaultAgentDeps                   *VaultAgentDeps
	Labels                           map[string]string
}

func (od *OperatorDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	key := helper.SuffixObjectKey(od.Core.Key, "operator")

	if _, err := od.VaultAgentDeps.Load(ctx, cl); err != nil {
		return false, err
	}

	od.WebhookCertificateControllerDeps = NewWebhookCertificateControllerDeps(key, od.Core)

	if _, err := od.WebhookCertificateControllerDeps.Load(ctx, cl); err != nil {
		return false, err
	}

	od.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	tn := od.Core.Object.Spec.Operator.TenantNamespace
	if tn != nil {
		od.TenantNamespace = corev1obj.NewNamespace(*tn)
	}

	od.Deployment = newOperatorDeployment(key, od.Core, od.VaultAgentDeps)
	od.WebhookService = newOperatorWebhookService(od.Deployment)
	od.ServiceAccount = corev1obj.NewServiceAccount(key)
	od.ClusterRole = rbacv1obj.NewClusterRole(key.Name)
	od.ClusterRoleBinding = rbacv1obj.NewClusterRoleBinding(key.Name)
	od.DelegateClusterRole = rbacv1obj.NewClusterRole(helper.SuffixObjectKey(key, "delegate").Name)

	if od.Core.Object.Spec.Operator.Standalone {
		od.DelegateClusterRoleBinding = rbacv1obj.NewClusterRoleBinding(od.DelegateClusterRole.Name)
	}

	od.WebhookConfig = admissionregistrationv1.NewMutatingWebhookConfiguration(key.Name)

	ok, err := lifecycle.Loaders{
		lifecycle.IgnoreNilLoader{Loader: od.TenantNamespace},
		od.OwnerConfigMap,
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
		lifecycle.IgnoreNilLoader{Loader: od.DelegateClusterRoleBinding},
		od.WebhookConfig,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (od *OperatorDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(od.OwnerConfigMap.Object, owner)
}

func (od *OperatorDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := od.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	// TODO Consider a Persister that does not error if the object already exists
	if od.TenantNamespace != nil {
		if err := od.TenantNamespace.Persist(ctx, cl); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	os := []lifecycle.Ownable{
		od.VaultAgentDeps,
		od.WebhookCertificateControllerDeps,
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
	}

	for _, o := range os {
		if err := od.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	objs := []lifecycle.Persister{
		od.VaultAgentDeps,
		od.WebhookConfig,
		od.WebhookCertificateControllerDeps,
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
		lifecycle.IgnoreNilPersister{Persister: od.DelegateClusterRoleBinding},
	}

	for _, obj := range objs {
		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (od *OperatorDeps) Configure(ctx context.Context) error {
	klog.Info("configuring operator deps")

	err := DependencyManager.SetDependencyOf(
		od.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: od.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		},
	)
	if err != nil {
		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		od.Deployment,
		od.WebhookService,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
		lifecycle.IgnoreNilLabelAnnotatableFrom{LabelAnnotatableFrom: od.DelegateClusterRoleBinding},
		od.ServiceAccount,
	}

	for _, laf := range lafs {
		for label, value := range od.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	objs := []obj.Configurable{
		od.VaultAgentDeps,
		od.WebhookCertificateControllerDeps,
		od.Deployment,
		od.WebhookService,
	}

	for _, obj := range objs {
		if err := obj.Configure(ctx); err != nil {
			return err
		}
	}

	ConfigureOperatorWebhookConfiguration(od, od.WebhookConfig)
	ConfigureOperatorClusterRole(od.ClusterRole)
	ConfigureClusterRoleBinding(od.ServiceAccount, od.ClusterRoleBinding)
	ConfigureOperatorDelegateClusterRole(od.DelegateClusterRole)

	if od.Core.Object.Spec.Operator.Standalone {
		ConfigureClusterRoleBinding(od.ServiceAccount, od.DelegateClusterRoleBinding)
	}

	return nil
}

func NewOperatorDeps(c *obj.Core) *OperatorDeps {
	return &OperatorDeps{
		Core:           c,
		VaultAgentDeps: NewVaultAgentDepsForRole(c.Object.Spec.Operator.VaultAgentRole, c),
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "operator",
			model.RelayAppInstanceLabel:   norm.AnyDNSLabelNameSuffixed("operator-", c.Key.Name),
			model.RelayAppComponentLabel:  "server",
			model.RelayAppManagedByLabel:  "relay-installer",
		},
	}
}
