package app

import (
	"context"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VaultConfigDeps struct {
	Core                         *obj.Core
	OwnerConfigMap               *corev1obj.ConfigMap
	JWTSigningKeyDeps            *JWTSigningKeyDeps
	VaultEngineConfigDeps        *VaultEngineConfigDeps
	VaultServerBuiltInConfigDeps *VaultServerBuiltInConfigDeps
}

func (vcd *VaultConfigDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	vcd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(vcd.Core.Key, "owner"))

	vcd.JWTSigningKeyDeps = NewJWTSigningKeyDeps(vcd.Core)

	if vcd.Core.Object.Spec.Vault.Server.BuiltIn != nil {
		vcd.VaultServerBuiltInConfigDeps = NewVaultServerBuiltInConfigDeps(vcd.Core)
	}

	vcd.VaultEngineConfigDeps = NewVaultSystemConfigDeps(vcd.Core, vcd.JWTSigningKeyDeps)

	ok, err := lifecycle.Loaders{
		vcd.OwnerConfigMap,
		vcd.JWTSigningKeyDeps,
		lifecycle.IgnoreNilLoader{Loader: vcd.VaultServerBuiltInConfigDeps},
		vcd.VaultEngineConfigDeps,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (vcd *VaultConfigDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(vcd.OwnerConfigMap.Object, owner)
}

func (vcd *VaultConfigDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vcd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	objs := []lifecycle.OwnablePersister{
		vcd.JWTSigningKeyDeps,
		lifecycle.IgnoreNilOwnablePersister{OwnablePersister: vcd.VaultServerBuiltInConfigDeps},
		vcd.VaultEngineConfigDeps,
	}

	for _, obj := range objs {
		if err := vcd.OwnerConfigMap.Own(ctx, obj); err != nil {
			return err
		}

		if err := obj.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (vcd *VaultConfigDeps) Configure(ctx context.Context) error {
	if err := DependencyManager.SetDependencyOf(
		vcd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: vcd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	objs := []obj.Configurable{
		vcd.JWTSigningKeyDeps,
		obj.IgnoreNilConfigurable{Configurable: vcd.VaultServerBuiltInConfigDeps},
		vcd.VaultEngineConfigDeps,
	}

	for _, obj := range objs {
		if err := obj.Configure(ctx); err != nil {
			return err
		}
	}

	return nil
}

func NewVaultConfigDeps(c *obj.Core) *VaultConfigDeps {
	return &VaultConfigDeps{
		Core: c,
	}
}
