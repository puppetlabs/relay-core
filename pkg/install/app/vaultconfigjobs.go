package app

import (
	"context"

	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type VaultConfigJobs struct {
	Core *obj.Core
	Auth *VaultConfigAuth

	UnsealJob *batchv1obj.Job
	ConfigJob *batchv1obj.Job

	key types.NamespacedName
}

var _ lifecycle.Loader = &VaultConfigJobs{}

func (vj *VaultConfigJobs) Load(ctx context.Context, cl client.Client) (bool, error) {
	vj.ConfigJob = batchv1obj.NewJob(vj.key)

	if _, ok := vj.Auth.UnsealKey(); ok {
		vj.UnsealJob = batchv1obj.NewJob(helper.SuffixObjectKey(vj.key, "unseal"))
	}

	ok, err := lifecycle.Loaders{
		lifecycle.IgnoreNilLoader{Loader: vj.UnsealJob},
		lifecycle.RequiredLoader{Loader: vj.ConfigJob},
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (vj *VaultConfigJobs) Persist(ctx context.Context, cl client.Client) error {
	ps := []lifecycle.Persister{
		lifecycle.IgnoreNilPersister{Persister: vj.UnsealJob},
		vj.ConfigJob,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (vj *VaultConfigJobs) Running() bool {
	running := func(job *batchv1obj.Job) bool {
		return job.Complete() || job.Failed()
	}

	if vj.UnsealJob != nil {
		if running(vj.UnsealJob) {
			return true
		}
	}

	return running(vj.ConfigJob)
}

func NewVaultConfigJobs(c *obj.Core, auth *VaultConfigAuth, key types.NamespacedName) *VaultConfigJobs {
	vj := &VaultConfigJobs{
		Core: c,
		Auth: auth,
		key:  key,
	}

	return vj
}

func ConfigureVaultConfigJobs(vj *VaultConfigJobs, cm *corev1obj.ConfigMap) {

}
