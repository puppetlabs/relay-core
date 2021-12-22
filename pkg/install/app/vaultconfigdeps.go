package app

import (
	"context"
	"fmt"

	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VaultConfigDepsLoadResult struct {
	JobsExist   bool
	JobsRunning bool
}

type VaultConfigDeps struct {
	Core *obj.Core

	Auth                 *VaultConfigAuth
	ConfigMap            *corev1obj.ConfigMap
	UnsealJob            *batchv1obj.Job
	ConfigJob            *batchv1obj.Job
	ServerServiceAccount *corev1obj.ServiceAccount
	OwnerConfigMap       *corev1obj.ConfigMap
}

func (vd *VaultConfigDeps) Load(ctx context.Context, cl client.Client) (*VaultConfigDepsLoadResult, error) {
	key := helper.SuffixObjectKey(vd.Core.Key, "vault-config")

	vd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	loaders := lifecycle.Loaders{
		vd.OwnerConfigMap,
		lifecycle.IgnoreNilLoader{Loader: vd.Auth},
	}

	if vd.Auth != nil {
		if vd.Core.Object.Spec.Vault.ConfigMapRef == nil {
			return &VaultConfigDepsLoadResult{}, fmt.Errorf("vault configuration enabled but the config map is missing")
		}

		vd.ConfigMap = corev1obj.NewConfigMap(client.ObjectKey{
			Namespace: vd.Core.Key.Namespace,
			Name:      vd.Core.Object.Spec.Vault.ConfigMapRef.Name,
		})

		vd.ConfigJob = batchv1obj.NewJob(key)
		if _, ok := vd.Auth.UnsealKeyEnvVar(); ok {
			vd.UnsealJob = batchv1obj.NewJob(helper.SuffixObjectKey(key, "unseal"))
		}

		vd.ServerServiceAccount = corev1obj.NewServiceAccount(
			client.ObjectKey{
				Name: vd.Core.Object.Spec.Vault.AuthDelegatorServiceAccountName,
			},
		)

		loaders = append(loaders,
			lifecycle.RequiredLoader{Loader: vd.ConfigMap},
			lifecycle.RequiredLoader{Loader: vd.ServerServiceAccount},
			vd.ConfigJob,
			vd.UnsealJob,
		)
	}

	// jobsExist, err := lifecycle.IgnoreNilLoader{Loader: vd.Jobs}.Load(ctx, cl)
	// if err != nil {
	// 	return &VaultConfigDepsLoadResult{}, err
	// }

	// _, err = lifecycle.Loaders{
	// 	vd.OwnerConfigMap,
	// 	lifecycle.IgnoreNilLoader{Loader: vd.Auth},
	// 	lifecycle.IgnoreNilLoader{Loader: vd.ConfigMap},
	// }.Load(ctx, cl)
	if _, err := loaders.Load(ctx, cl); err != nil {
		return &VaultConfigDepsLoadResult{}, err
	}

	jobsRunning := false
	if vd.Auth != nil {
		jobsRunning = vd.Running()
	}

	return &VaultConfigDepsLoadResult{JobsExist: false, JobsRunning: jobsRunning}, nil
}

func (vd *VaultConfigDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	var os []lifecycle.Ownable
	if vd.Auth != nil {
		os = []lifecycle.Ownable{
			vd.ConfigJob,
			lifecycle.IgnoreNilOwnable{Ownable: vd.UnsealJob},
		}
	}

	for _, o := range os {
		if err := vd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	ps := []lifecycle.Persister{
		lifecycle.IgnoreNilPersister{Persister: vd.UnsealJob},
		lifecycle.IgnoreNilPersister{Persister: vd.ConfigJob},
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (vd *VaultConfigDeps) Running() bool {
	running := func(job *batchv1obj.Job) bool {
		if job.Object.GetUID() == "" {
			return false
		}

		return !job.Complete() && !job.Failed()
	}

	if vd.UnsealJob != nil {
		if running(vd.UnsealJob) {
			return true
		}
	}

	return running(vd.ConfigJob)
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

func ConfigureVaultConfigDeps(vd *VaultConfigDeps) error {
	if err := DependencyManager.SetDependencyOf(
		vd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: vd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	ConfigureVaultConfigJob(vd, vd.ConfigJob)
	ConfigureVaultConfigUnsealJob(vd, vd.UnsealJob)

	return nil
}
