package obj_test

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/operator/obj"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var e2e *testutil.EndToEndEnvironment

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

func Client(t *testing.T) client.Client {
	require.NotNil(t, e2e)
	return e2e.ControllerRuntimeClient
}

func WithTestNamespace(t *testing.T, ctx context.Context, fn func(ns *obj.Namespace)) {
	require.NotNil(t, e2e)
	e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
		fn(&obj.Namespace{Name: ns.GetName(), Object: ns})
	})
}

func TestMain(m *testing.M) {
	os.Exit(testutil.RunEndToEnd(m, func(e *testutil.EndToEndEnvironment) {
		e2e = e
	}))
}
