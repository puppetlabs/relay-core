package e2e_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/puppetlabs/leg/k8sutil/pkg/test/endtoend"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TODO: If the test were passed a specific RelayCore, it could look these
// constants up dynamically.

const (
	TestSystemNamespace            = "relay-system"
	TestVaultEngineTenantPath      = "customers"
	TestVaultServiceName           = "vault"
	TestVaultCredentialsSecretName = "vault"
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
	kfs := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(kfs)
	_ = kfs.Set("v", "5")

	log.SetLogger(klogr.NewWithOptions(klogr.WithFormat(klogr.FormatKlog)))
}

type EnvironmentInTest struct {
	*endtoend.Environment
	Labels map[string]string
	t      *testing.T
	nf     endtoend.NamespaceFactory
}

func (eit *EnvironmentInTest) WithNamespace(ctx context.Context, fn func(ns *corev1.Namespace)) {
	require.NoError(eit.t, endtoend.WithNamespace(ctx, eit.Environment, eit.nf, func(ns *corev1.Namespace) {
		defer CleanUp(eit.t, eit, ns)
		fn(ns)
	}))
}

func WithEnvironmentInTest(t *testing.T, fn func(eit *EnvironmentInTest)) {
	viper.SetEnvPrefix("relay_test_e2e")
	viper.AutomaticEnv()

	kubeconfigs := strings.TrimSpace(viper.GetString("kubeconfig"))
	if testing.Short() {
		t.Skip("not running end-to-end tests with -short")
	} else if kubeconfigs == "" {
		t.Skip("not running end-to-end tests without one or more Kubeconfigs specified by RELAY_TEST_E2E_KUBECONFIG")
	}

	opts := []endtoend.EnvironmentOption{
		endtoend.EnvironmentWithClientScheme(dependency.Scheme),
		endtoend.EnvironmentWithClientKubeconfigs(filepath.SplitList(kubeconfigs)),
		endtoend.EnvironmentWithClientContext(viper.GetString("context")),
	}

	require.NoError(t, endtoend.WithEnvironment(opts, func(e *endtoend.Environment) {
		ls := map[string]string{
			"e2e.tests.relay.sh/harness":   "end-to-end",
			"e2e.tests.relay.sh/test.hash": testHash(t),
		}

		eit := &EnvironmentInTest{
			Environment: e,
			Labels:      ls,
			t:           t,
			nf:          endtoend.NewTestNamespaceFactory(t, endtoend.NamespaceWithLabels(ls)),
		}

		fn(eit)
	}))
}

func WithNamespacedEnvironmentInTest(t *testing.T, ctx context.Context, fn func(eit *EnvironmentInTest, ns *corev1.Namespace)) {
	WithEnvironmentInTest(t, func(eit *EnvironmentInTest) {
		eit.WithNamespace(ctx, func(ns *corev1.Namespace) {
			fn(eit, ns)
		})
	})
}

func testHash(t *testing.T) string {
	h := sha256.Sum256([]byte(t.Name()))
	return hex.EncodeToString(h[:])[:63]
}
