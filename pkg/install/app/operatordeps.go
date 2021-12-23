package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/admissionregistrationv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatorDeps struct {
	Core                             *obj.Core
	Deployment                       *operatorDeployment
	WebhookService                   *operatorWebhookService
	ServiceAccount                   *corev1.ServiceAccount
	ClusterRole                      *rbacv1.ClusterRole
	ClusterRoleBinding               *rbacv1.ClusterRoleBinding
	DelegateClusterRole              *rbacv1.ClusterRole
	WebhookConfig                    *admissionregistrationv1.MutatingWebhookConfiguration
	OwnerConfigMap                   *corev1.ConfigMap
	WebhookCertificateControllerDeps *WebhookCertificateControllerDeps
	VaultAgentDeps                   *VaultAgentDeps
	Labels                           map[string]string
}

func (od *OperatorDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := od.Core.Load(ctx, cl); err != nil {
		return false, err
	}

	key := helper.SuffixObjectKey(od.Core.Key, "operator")

	if _, err := od.VaultAgentDeps.Load(ctx, cl); err != nil {
		return false, err
	}

	od.WebhookCertificateControllerDeps = NewWebhookCertificateControllerDeps(key, od.Core)

	if _, err := od.WebhookCertificateControllerDeps.Load(ctx, cl); err != nil {
		return false, err
	}

	od.OwnerConfigMap = corev1.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	od.Deployment = newOperatorDeployment(key, od.Core, od.VaultAgentDeps)
	od.WebhookService = newOperatorWebhookService(od.Deployment)
	od.ServiceAccount = corev1.NewServiceAccount(key)
	od.ClusterRole = rbacv1.NewClusterRole(key.Name)
	od.ClusterRoleBinding = rbacv1.NewClusterRoleBinding(key.Name)
	od.DelegateClusterRole = rbacv1.NewClusterRole(helper.SuffixObjectKey(key, "delegate").Name)
	od.WebhookConfig = admissionregistrationv1.NewMutatingWebhookConfiguration(key.Name)

	ok, err := lifecycle.Loaders{
		od.OwnerConfigMap,
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
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

	objs := []lifecycle.OwnablePersister{
		od.VaultAgentDeps,
		od.WebhookConfig,
		od.WebhookCertificateControllerDeps,
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
	}

	for _, obj := range objs {
		if err := od.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

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
		od.ServiceAccount,
	}

	for _, laf := range lafs {
		for label, value := range od.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	objs := []Configurable{
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
	ConfigureClusterRoleBinding(od.Core, od.ClusterRoleBinding)
	ConfigureOperatorDelegateClusterRole(od.DelegateClusterRole)

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
			model.RelayAppManagedByLabel:  "relay-install-operator",
		},
	}
}
