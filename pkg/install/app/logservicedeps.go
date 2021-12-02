package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LogServiceDeps struct {
	Core           *obj.Core
	Deployment     *appsv1.Deployment
	Service        *corev1.Service
	ServiceAccount *corev1.ServiceAccount
	OwnerConfigMap *corev1.ConfigMap
	VaultAgentDeps *VaultAgentDeps
	Labels         map[string]string
}

func (ld *LogServiceDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	if _, err := ld.Core.Load(ctx, cl); err != nil {
		return false, err
	}

	if _, err := ld.VaultAgentDeps.Load(ctx, cl); err != nil {
		return false, err
	}

	key := helper.SuffixObjectKey(ld.Core.Key, "log-service")

	ld.OwnerConfigMap = corev1.NewConfigMap(helper.SuffixObjectKey(key, "owner"))
	ld.Deployment = appsv1.NewDeployment(key)
	ld.Service = corev1.NewService(key)
	ld.ServiceAccount = corev1.NewServiceAccount(key)

	ok, err := lifecycle.Loaders{
		ld.OwnerConfigMap,
		ld.Deployment,
		ld.Service,
		ld.ServiceAccount,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (ld *LogServiceDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := ld.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		ld.Deployment,
		ld.Service,
		ld.ServiceAccount,
	}

	for _, o := range os {
		if err := ld.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		ld.VaultAgentDeps,
		ld.Deployment,
		ld.Service,
		ld.ServiceAccount,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewLogServiceDeps(c *obj.Core) *LogServiceDeps {
	return &LogServiceDeps{
		Core:           c,
		VaultAgentDeps: NewVaultAgentDepsForRole(obj.VaultAgentRoleLogService, c),
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "log-service",
			model.RelayAppInstanceLabel:   "log-service-" + c.Key.Name,
			model.RelayAppComponentLabel:  "server",
			model.RelayAppManagedByLabel:  "relay-install-operator",
		},
	}
}

func ConfigureLogServiceDeps(ctx context.Context, ld *LogServiceDeps) error {
	if err := DependencyManager.SetDependencyOf(
		ld.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: ld.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		ld.Deployment,
		ld.Service,
		ld.ServiceAccount,
	}

	for _, laf := range lafs {
		for label, value := range ld.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	ConfigureVaultAgentDeps(ld.VaultAgentDeps)

	ConfigureLogServiceDeployment(ld, ld.Deployment)
	ConfigureLogServiceService(ld, ld.Service)

	return nil
}
