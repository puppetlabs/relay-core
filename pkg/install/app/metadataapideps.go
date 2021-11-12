package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
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
}

func (md *MetadataAPIDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if ok, err := md.Core.Load(ctx, cl); err != nil {
		return false, err
	} else if !ok {
		return ok, nil
	}

	if ok, err := md.VaultAgentDeps.Load(ctx, cl); err != nil {
		return false, err
	} else if !ok {
		return ok, nil
	}

	key := md.Core.Key

	md.OwnerConfigMap = corev1.NewConfigMap(SuffixObjectKey(key, "owner"))

	md.Deployment = appsv1.NewDeployment(SuffixObjectKey(key, "metadata-api"))
	md.Service = corev1.NewService(SuffixObjectKey(key, "metadata-api"))
	md.ServiceAccount = corev1.NewServiceAccount(SuffixObjectKey(key, "metadata-api"))
	md.ClusterRole = rbacv1.NewClusterRole(SuffixObjectKey(key, "metadata-api").Name)
	md.ClusterRoleBinding = rbacv1.NewClusterRoleBinding(SuffixObjectKey(key, "metadata-api").Name)

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
		md.ClusterRole,
		md.ClusterRoleBinding,
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
	}
}

func ConfigureMetadataAPIDeps(md *MetadataAPIDeps) error {
	if err := DependencyManager.SetDependencyOf(
		md.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: md.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	// ConfigureDeploymentWithVaultAgent(md.VaultAgentDeps, md.Deployment)

	return nil
}
