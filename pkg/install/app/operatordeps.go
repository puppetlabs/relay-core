package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/admissionregistrationv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type OperatorDeps struct {
	Core                  *obj.Core
	Deployment            *appsv1.Deployment
	WebhookService        *corev1.Service
	ServiceAccount        *corev1.ServiceAccount
	ClusterRole           *rbacv1.ClusterRole
	ClusterRoleBinding    *rbacv1.ClusterRoleBinding
	DelegateClusterRole   *rbacv1.ClusterRole
	MutatingWebhookConfig *admissionregistrationv1.MutatingWebhookConfiguration
	OwnerConfigMap        *corev1.ConfigMap
	VaultAgentDeps        *VaultAgentDeps
}

func (od *OperatorDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if ok, err := od.Core.Load(ctx, cl); err != nil {
		return false, err
	} else if !ok {
		return ok, nil
	}

	if ok, err := od.VaultAgentDeps.Load(ctx, cl); err != nil {
		return false, err
	} else if !ok {
		return ok, nil
	}

	key := od.Core.Key

	od.OwnerConfigMap = corev1.NewConfigMap(SuffixObjectKey(key, "owner"))

	od.Deployment = appsv1.NewDeployment(SuffixObjectKey(key, "operator"))
	od.WebhookService = corev1.NewService(SuffixObjectKey(key, "operator-webhook"))
	od.ServiceAccount = corev1.NewServiceAccount(SuffixObjectKey(key, "operator"))
	od.ClusterRole = rbacv1.NewClusterRole(SuffixObjectKey(key, "operator").Name)
	od.ClusterRoleBinding = rbacv1.NewClusterRoleBinding(SuffixObjectKey(key, "operator").Name)
	od.DelegateClusterRole = rbacv1.NewClusterRole(SuffixObjectKey(key, "operator-delegate").Name)
	od.MutatingWebhookConfig = admissionregistrationv1.NewMutatingWebhookConfiguration(SuffixObjectKey(key, "operator").Name)

	ok, err := lifecycle.Loaders{
		od.OwnerConfigMap,
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
		od.MutatingWebhookConfig,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (od *OperatorDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := od.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
		od.MutatingWebhookConfig,
	}
	for _, o := range os {
		if err := od.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		od.Deployment,
		od.WebhookService,
		od.ServiceAccount,
		od.ClusterRole,
		od.ClusterRoleBinding,
		od.DelegateClusterRole,
		od.MutatingWebhookConfig,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewOperatorDeps(c *obj.Core, vd *VaultAgentDeps) *OperatorDeps {
	return &OperatorDeps{
		Core:           c,
		VaultAgentDeps: vd,
	}
}

func ConfigureOperator(ctx context.Context, od *OperatorDeps) error {
	if err := DependencyManager.SetDependencyOf(
		od.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: od.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	return nil
}
