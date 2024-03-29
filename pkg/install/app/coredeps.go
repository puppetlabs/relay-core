package app

import (
	"context"
	"fmt"
	"net/url"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CoreDepsLoadResult struct {
	All bool
}

type CoreDeps struct {
	Core            *obj.Core
	OwnerConfigMap  *corev1obj.ConfigMap
	Namespace       *corev1obj.Namespace
	VaultConfigDeps *VaultConfigDeps
	OperatorDeps    *OperatorDeps
	MetadataAPIDeps *MetadataAPIDeps
	LogServiceDeps  *LogServiceDeps
}

func (cd *CoreDeps) Delete(ctx context.Context, cl client.Client, opts ...lifecycle.DeleteOption) (bool, error) {
	if cd.OwnerConfigMap == nil || cd.OwnerConfigMap.Object.GetUID() == "" {
		return true, nil
	}

	if ok, err := DependencyManager.IsDependencyOf(
		cd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: cd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {
		return false, err
	} else if ok {
		return cd.OwnerConfigMap.Delete(ctx, cl, opts...)
	}

	return true, nil
}

func (cd *CoreDeps) Load(ctx context.Context, cl client.Client) (*CoreDepsLoadResult, error) {
	if _, err := cd.Core.Load(ctx, cl); err != nil {
		return nil, err
	}

	cd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(cd.Core.Key, "owner"))

	cd.VaultConfigDeps = NewVaultConfigDeps(cd.Core)

	cd.OperatorDeps = NewOperatorDeps(cd.Core, cd.VaultConfigDeps)
	cd.MetadataAPIDeps = NewMetadataAPIDeps(cd.Core)

	if cd.Core.Object.Spec.LogService != nil {
		cd.LogServiceDeps = NewLogServiceDeps(cd.Core)
	}

	ok, err := lifecycle.Loaders{
		lifecycle.RequiredLoader{Loader: cd.Namespace},
		cd.OwnerConfigMap,
		cd.VaultConfigDeps,
		cd.OperatorDeps,
		cd.MetadataAPIDeps,
		lifecycle.IgnoreNilLoader{Loader: cd.LogServiceDeps},
	}.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	return &CoreDepsLoadResult{All: ok}, nil
}

func (cd *CoreDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := cd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	if err := cd.Namespace.Persist(ctx, cl); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		cd.VaultConfigDeps,
		cd.MetadataAPIDeps,
		cd.OperatorDeps,
		lifecycle.IgnoreNilOwnablePersister{OwnablePersister: cd.LogServiceDeps},
	}

	for _, obj := range objs {
		if err := cd.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (cd *CoreDeps) Configure(_ context.Context) error {
	if err := DependencyManager.SetDependencyOf(
		cd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: cd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	ConfigureCoreDefaults(cd)

	return nil
}

func NewCoreDeps(c *obj.Core) *CoreDeps {
	return &CoreDeps{
		Core:      c,
		Namespace: corev1obj.NewNamespace(c.Object.GetNamespace()),
	}
}

func ApplyCoreDeps(ctx context.Context, cl client.Client, c *obj.Core) (*CoreDeps, error) {
	cd := NewCoreDeps(c)

	_, err := cd.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	if err := cd.Configure(ctx); err != nil {
		return nil, err
	}

	if err := cd.Core.Persist(ctx, cl); err != nil {
		return nil, err
	}

	objs := []obj.Configurable{
		cd.VaultConfigDeps,
		cd.OperatorDeps,
		cd.MetadataAPIDeps,
		obj.IgnoreNilConfigurable{Configurable: cd.LogServiceDeps},
	}

	for _, obj := range objs {
		if err := obj.Configure(ctx); err != nil {
			return nil, err
		}
	}

	klog.Info("persisting all deps")
	if err := cd.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return cd, nil
}

func ConfigureCoreDefaults(cd *CoreDeps) {
	core := cd.Core

	if core.Object.Spec.Operator.ServiceAccountName == "" {
		core.Object.Spec.Operator.ServiceAccountName = cd.OperatorDeps.ServiceAccount.Key.Name
	}

	if core.Object.Spec.MetadataAPI.ServiceAccountName == "" {
		core.Object.Spec.MetadataAPI.ServiceAccountName = cd.MetadataAPIDeps.ServiceAccount.Key.Name
	}

	if core.Object.Spec.MetadataAPI.URL == nil {
		u := url.URL{
			Scheme: "http",
		}

		if core.Object.Spec.MetadataAPI.TLSSecretName != nil {
			u.Scheme = "https"
		}

		u.Host = fmt.Sprintf("%s.%s.svc.cluster.local", cd.MetadataAPIDeps.Service.Key.Name, cd.MetadataAPIDeps.Service.Key.Namespace)

		us := u.String()
		core.Object.Spec.MetadataAPI.URL = &us
	}

	if core.Object.Spec.LogService != nil {
		if core.Object.Spec.MetadataAPI.LogServiceURL == nil {
			logServiceURL := fmt.Sprintf("%s.%s:%d",
				cd.LogServiceDeps.Service.Key.Name, cd.LogServiceDeps.Service.Key.Namespace,
				DefaultLogServicePort)
			core.Object.Spec.MetadataAPI.LogServiceURL = &logServiceURL
		}
	}
}
