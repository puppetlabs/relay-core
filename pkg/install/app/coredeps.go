package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CoreDepsLoadResult struct {
	All bool
}

type CoreDeps struct {
	Core            *obj.Core
	Namespace       *corev1.Namespace
	OperatorDeps    *OperatorDeps
	MetadataAPIDeps *MetadataAPIDeps
	LogServiceDeps  *LogServiceDeps
}

func (cd *CoreDeps) Load(ctx context.Context, cl client.Client) (*CoreDepsLoadResult, error) {
	if _, err := cd.Core.Load(ctx, cl); err != nil {
		return nil, err
		// } else if !ok {
		// 	return &CoreDepsLoadResult{}, nil
	}

	cd.OperatorDeps = NewOperatorDeps(cd.Core)
	cd.MetadataAPIDeps = NewMetadataAPIDeps(cd.Core)
	cd.LogServiceDeps = NewLogServiceDeps(cd.Core)

	ok, err := lifecycle.Loaders{
		lifecycle.RequiredLoader{Loader: cd.Namespace},
		cd.OperatorDeps,
		cd.MetadataAPIDeps,
		cd.LogServiceDeps,
	}.Load(ctx, cl)
	if err != nil {
		return nil, err
	}

	return &CoreDepsLoadResult{All: ok}, nil
}

func (cd *CoreDeps) Persist(ctx context.Context, cl client.Client) error {
	ps := []lifecycle.Persister{
		cd.Namespace,
		cd.MetadataAPIDeps,
		cd.OperatorDeps,
		cd.LogServiceDeps,
	}

	for _, p := range ps {
		if err := p.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewCoreDeps(c *obj.Core) *CoreDeps {
	return &CoreDeps{
		Core:      c,
		Namespace: corev1.NewNamespace(c.Object.GetNamespace()),
	}
}

func ApplyCoreDeps(ctx context.Context, cl client.Client, c *obj.Core) (*CoreDeps, error) {
	cd := NewCoreDeps(c)

	_, err := cd.Load(ctx, cl)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	// if !result.All {
	// 	err := fmt.Errorf("waiting for upstream dependencies")
	// 	klog.Error(err)

	// 	return nil, err
	// }

	ConfigureCore(cd)

	if err := ConfigureOperatorDeps(ctx, cd.OperatorDeps); err != nil {
		return nil, err
	}

	if err := ConfigureMetadataAPIDeps(ctx, cd.MetadataAPIDeps); err != nil {
		return nil, err
	}

	if err := ConfigureLogServiceDeps(ctx, cd.LogServiceDeps); err != nil {
		return nil, err
	}

	if err := cd.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return cd, nil
}
