package app

import (
	"context"
	"fmt"
	"net/url"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
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

	ConfigureCoreDeps(cd)

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

func ConfigureCoreDeps(cd *CoreDeps) {
	core := cd.Core

	if core.Object.Spec.Operator == nil {
		core.Object.Spec.Operator = &v1alpha1.OperatorConfig{}
	}

	if core.Object.Spec.MetadataAPI == nil {
		core.Object.Spec.MetadataAPI = &v1alpha1.MetadataAPIConfig{}
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

	if core.Object.Spec.LogService.CredentialsSecretName == "" {
		core.Object.Spec.LogService.CredentialsSecretName = SuffixObjectKey(
			cd.LogServiceDeps.Deployment.Key,
			"google-application-credentials",
		).Name
	}
}
