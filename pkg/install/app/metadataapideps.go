package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetadataAPIDeps struct {
	Core               *obj.Core
	Deployment         *appsv1.Deployment
	Service            *corev1.Service
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

	key := SuffixObjectKey(md.Core.Key, "metadata-api")

	md.OwnerConfigMap = corev1.NewConfigMap(SuffixObjectKey(key, "owner"))

	md.Deployment = appsv1.NewDeployment(key)
	md.Service = corev1.NewService(key)
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

func (md *MetadataAPIDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := md.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		md.Deployment,
		md.Service,
		md.ServiceAccount,
	}

	for _, o := range os {
		if err := md.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		md.VaultAgentDeps,
		md.Deployment,
		md.Service,
		md.ServiceAccount,
		md.ClusterRole,
		md.ClusterRoleBinding,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewMetadataAPIDeps(c *obj.Core) *MetadataAPIDeps {
	return &MetadataAPIDeps{
		Core:           c,
		VaultAgentDeps: NewVaultAgentDepsForRole(obj.VaultAgentRoleMetadataAPI, c),
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "metadata-api",
			model.RelayAppInstanceLabel:   "metadata-api-" + c.Key.Name,
			model.RelayAppComponentLabel:  "server",
			model.RelayAppManagedByLabel:  "relay-install-operator",
		},
	}
}

func ConfigureMetadataAPIDeps(ctx context.Context, md *MetadataAPIDeps) error {
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

	ConfigureVaultAgentDeps(md.VaultAgentDeps)

	ConfigureMetadataAPIDeployment(md, md.Deployment)
	ConfigureMetadataAPIClusterRole(md.ClusterRole)
	ConfigureClusterRoleBinding(md.Core, md.ClusterRoleBinding)
	ConfigureMetadataAPIService(md, md.Service)

	return nil
}
