package app

import (
	"context"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WebhookCertificateControllerDeps struct {
	Core               *obj.Core
	TargetDeployment   types.NamespacedName
	Deployment         *appsv1obj.Deployment
	ServiceAccount     *corev1obj.ServiceAccount
	ClusterRole        *rbacv1obj.ClusterRole
	ClusterRoleBinding *rbacv1obj.ClusterRoleBinding
	OwnerConfigMap     *corev1obj.ConfigMap
	Labels             map[string]string
}

func (d *WebhookCertificateControllerDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := d.Core.Load(ctx, cl); err != nil {
		return false, err
	}

	key := helper.SuffixObjectKey(d.Core.Key, "webhook-certificate-controller")

	d.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))
	d.Deployment = appsv1obj.NewDeployment(key)
	d.ServiceAccount = corev1obj.NewServiceAccount(key)
	d.ClusterRole = rbacv1obj.NewClusterRole(key.Name)
	d.ClusterRoleBinding = rbacv1obj.NewClusterRoleBinding(key.Name)

	ok, err := lifecycle.Loaders{
		d.OwnerConfigMap,
		d.Deployment,
		d.ServiceAccount,
		d.ClusterRole,
		d.ClusterRoleBinding,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (d *WebhookCertificateControllerDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(d.OwnerConfigMap.Object, owner)
}

func (d *WebhookCertificateControllerDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := d.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		d.Deployment,
		d.ServiceAccount,
		d.ClusterRole,
		d.ClusterRoleBinding,
	}

	for _, obj := range objs {
		if err := d.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (wd *WebhookCertificateControllerDeps) Configure(ctx context.Context) error {
	klog.Info("configuring webhook certificate controller deps")

	if err := DependencyManager.SetDependencyOf(
		wd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: wd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		wd.Deployment,
		wd.ServiceAccount,
		wd.ClusterRole,
		wd.ClusterRoleBinding,
	}

	for _, laf := range lafs {
		for label, value := range wd.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	ConfigureWebhookCertificateControllerDeployment(wd, wd.Deployment)
	ConfigureWebhookCertificateControllerClusterRole(wd.ClusterRole)
	ConfigureWebhookCertificateControllerClusterRoleBinding(wd.Core, wd.ClusterRoleBinding)

	return nil
}

func NewWebhookCertificateControllerDeps(target types.NamespacedName, c *obj.Core) *WebhookCertificateControllerDeps {
	return &WebhookCertificateControllerDeps{
		Core:             c,
		TargetDeployment: target,
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "operator",
			model.RelayAppInstanceLabel:   "operator-" + c.Key.Name,
			model.RelayAppComponentLabel:  "webhook-certificate-server",
			model.RelayAppManagedByLabel:  "relay-install-operator",
		},
	}
}
