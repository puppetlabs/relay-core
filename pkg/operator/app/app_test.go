package app_test

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	TestIssuer = authenticate.IssuerFunc(func(ctx context.Context, claims *authenticate.Claims) (authenticate.Raw, error) {
		tok, err := json.Marshal(claims)
		if err != nil {
			return nil, err
		}

		return authenticate.Raw(tok), nil
	})
	TestMetadataAPIURL = &url.URL{Scheme: "http", Host: "stub.example.com"}
)

func init() {
	log.SetLogger(klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog)))
}
