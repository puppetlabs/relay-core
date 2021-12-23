package app

import (
	"context"

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

type MetadataAPIDeps struct {
	Core               *obj.Core
	Deployment         *metadataAPIDeployment
	Service            *metadataAPIService
	ServiceAccount     *corev1.ServiceAccount
	ClusterRole        *rbacv1.ClusterRole
	ClusterRoleBinding *rbacv1.ClusterRoleBinding
	OwnerConfigMap     *corev1.ConfigMap
	VaultAgentDeps     *VaultAgentDeps
	Labels             map[string]string
}

func (md *MetadataAPIDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := md.Core.Load(ctx, cl); err != nil {
		return false, err
	}

	if _, err := md.VaultAgentDeps.Load(ctx, cl); err != nil {
		return false, err
	}

	key := helper.SuffixObjectKey(md.Core.Key, "metadata-api")

	md.OwnerConfigMap = corev1.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	md.Deployment = newMetadataAPIDeployment(key, md.Core, md.VaultAgentDeps)
	md.Service = newMetadataAPIService(md.Core, md.Deployment)
	md.ServiceAccount = corev1.NewServiceAccount(key)
	md.ClusterRole = rbacv1.NewClusterRole(key.Name)
	md.ClusterRoleBinding = rbacv1.NewClusterRoleBinding(key.Name)

	ok, err := lifecycle.Loaders{
		md.OwnerConfigMap,
		md.Deployment,
		md.Service,
		md.ServiceAccount,
		md.ClusterRole,
		md.ClusterRoleBinding,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (md *MetadataAPIDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(md.OwnerConfigMap.Object, owner)
}

func (md *MetadataAPIDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := md.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		md.VaultAgentDeps,
		md.Deployment,
		md.Service,
		md.ServiceAccount,
		md.ClusterRole,
		md.ClusterRoleBinding,
	}

	for _, obj := range objs {
		if err := md.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (md *MetadataAPIDeps) Configure(ctx context.Context) error {
	klog.Info("configuring metadata-api deps")

	if err := DependencyManager.SetDependencyOf(
		md.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: md.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		md.Deployment,
		md.Service,
		md.ClusterRole,
		md.ClusterRoleBinding,
		md.ServiceAccount,
	}

	for _, laf := range lafs {
		for label, value := range md.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	objs := []Configurable{
		md.VaultAgentDeps,
		md.Deployment,
		md.Service,
	}

	for _, obj := range objs {
		if err := obj.Configure(ctx); err != nil {
			return err
		}
	}

	ConfigureMetadataAPIClusterRole(md.ClusterRole)
	ConfigureClusterRoleBinding(md.Core, md.ClusterRoleBinding)

	return nil
}

func NewMetadataAPIDeps(c *obj.Core) *MetadataAPIDeps {
	return &MetadataAPIDeps{
		Core:           c,
		VaultAgentDeps: NewVaultAgentDepsForRole(c.Object.Spec.MetadataAPI.VaultAgentRole, c),
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "metadata-api",
			model.RelayAppInstanceLabel:   norm.AnyDNSLabelNameSuffixed("metadata-api-", c.Key.Name),
			model.RelayAppComponentLabel:  "server",
			model.RelayAppManagedByLabel:  "relay-install-operator",
		},
	}
}
