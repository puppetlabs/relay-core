package app

import (
	"context"

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

	vd.ConfigMap = corev1obj.NewConfigMap(key)

	if vd.Auth != nil {
		vd.Jobs = NewVaultConfigJobs(vd.Core, vd.Auth, key)
	}

	jobsExist, err := lifecycle.IgnoreNilLoader{Loader: vd.Jobs}.Load(ctx, cl)
	if err != nil {
		return &VaultConfigDepsLoadResult{}, err
	}

	_, err = lifecycle.Loaders{
		vd.OwnerConfigMap,
		lifecycle.IgnoreNilLoader{Loader: vd.Auth},
		vd.ConfigMap,
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

	os := []lifecycle.Ownable{
		vd.ConfigMap,
	}
	if vd.Jobs != nil {
		os = append(os,
			vd.Jobs.ConfigJob,
			lifecycle.IgnoreNilOwnable{Ownable: vd.Jobs.UnsealJob},
		)
	}

	for _, o := range os {
		if err := vd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		vd.ConfigMap,
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
	ConfigureVaultConfigConfigMap(vd.Core, vd.ConfigMap)
	ConfigureVaultConfigJobs(vd.Jobs, vd.ConfigMap)

	return nil
}
