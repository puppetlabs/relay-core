package app

import (
	"context"
	"fmt"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VaultConfigDepsLoadResult struct {
	JobsExist   bool
	JobsRunning bool
}

type VaultConfigDeps struct {
	Core *obj.Core

	Auth           *VaultConfigAuth
	ConfigMap      *corev1obj.ConfigMap
	Jobs           *VaultConfigJobs
	OwnerConfigMap *corev1obj.ConfigMap
}

func (vd *VaultConfigDeps) Load(ctx context.Context, cl client.Client) (*VaultConfigDepsLoadResult, error) {
	key := helper.SuffixObjectKey(vd.Core.Key, "vault-config")

	vd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	if vd.Auth != nil {
		if vd.Core.Object.Spec.Vault.ConfigMapRef == nil {
			return &VaultConfigDepsLoadResult{}, fmt.Errorf("vault configuration enabled but the config map is missing")
		}

		vd.ConfigMap = corev1obj.NewConfigMap(client.ObjectKey{
			Namespace: vd.Core.Key.Namespace,
			Name:      vd.Core.Object.Spec.Vault.ConfigMapRef.Name,
		})
		vd.Jobs = NewVaultConfigJobs(vd.Core, vd.Auth, key)
	}

	jobsExist, err := lifecycle.IgnoreNilLoader{Loader: vd.Jobs}.Load(ctx, cl)
	if err != nil {
		return &VaultConfigDepsLoadResult{}, err
	}

	_, err = lifecycle.Loaders{
		vd.OwnerConfigMap,
		lifecycle.IgnoreNilLoader{Loader: vd.Auth},
		lifecycle.IgnoreNilLoader{Loader: vd.ConfigMap},
	}.Load(ctx, cl)
	if err != nil {
		return &VaultConfigDepsLoadResult{}, err
	}

	jobsRunning := false
	if vd.Jobs != nil {
		jobsRunning = vd.Jobs.Running()
	}

	return &VaultConfigDepsLoadResult{JobsExist: jobsExist, JobsRunning: jobsRunning}, nil
}

func (vd *VaultConfigDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	var os []lifecycle.Ownable
	if vd.Jobs != nil {
		os = []lifecycle.Ownable{
			vd.Jobs.ConfigJob,
			lifecycle.IgnoreNilOwnable{Ownable: vd.Jobs.UnsealJob},
		}
	}

	for _, o := range os {
		if err := vd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		lifecycle.IgnoreNilPersister{Persister: vd.Jobs},
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewVaultConfigDeps(c *obj.Core) *VaultConfigDeps {
	vd := &VaultConfigDeps{
		Core: c,
	}

	if c.Object.Spec.Vault.Auth != nil {
		vd.Auth = NewVaultConfigAuth(c, c.Object.Spec.Vault.Auth)
	}

	return vd
}

func ConfigureVaultConfigDeps(ctx context.Context, vd *VaultConfigDeps) error {
	ConfigureVaultConfigJobs(vd.Jobs, vd.ConfigMap)

	return nil
}
