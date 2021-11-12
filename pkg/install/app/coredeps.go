package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/obj"
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
}

func (cd *CoreDeps) Load(ctx context.Context, cl client.Client) (*CoreDepsLoadResult, error) {
	if ok, err := cd.Core.Load(ctx, cl); err != nil {
		return nil, err
	} else if !ok {
		return &CoreDepsLoadResult{}, nil
	}

	cd.OperatorDeps = NewOperatorDeps(cd.Core)
	cd.MetadataAPIDeps = NewMetadataAPIDeps(cd.Core)

	ok, err := lifecycle.Loaders{
		lifecycle.RequiredLoader{Loader: cd.Namespace},
		cd.OperatorDeps,
		cd.MetadataAPIDeps,
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

	if _, err := cd.Load(ctx, cl); err != nil {
		return nil, err
	}

	ConfigureCore(cd)

	if err := ConfigureOperatorDeps(cd.OperatorDeps); err != nil {
		return nil, err
	}

	if err := ConfigureMetadataAPIDeps(cd.MetadataAPIDeps); err != nil {
		return nil, err
	}

	if err := cd.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return cd, nil
}
