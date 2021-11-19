package app

import (
	"fmt"
	"net/url"

	"github.com/davecgh/go-spew/spew"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
)

func ConfigureCore(cd *CoreDeps) {
	core := cd.Core

	spew.Dump(cd)

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
}
