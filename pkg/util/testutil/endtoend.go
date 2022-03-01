package testutil

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/puppetlabs/leg/k8sutil/pkg/test/endtoend"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type EndToEndEnvironment struct {
	*endtoend.Environment
	Labels                 map[string]string
	LabelSelector          metav1.LabelSelector
	TektonInterface        tekton.Interface
	GVisorRuntimeClassName string
	t                      *testing.T
	nf                     endtoend.NamespaceFactory
	unf                    endtoend.NamespaceFactory
}

func (e *EndToEndEnvironment) WithTestNamespace(ctx context.Context, fn func(ns *corev1.Namespace)) {
	require.NoError(e.t, endtoend.WithNamespace(ctx, e.Environment, e.nf, fn))
}

// WithUtilNamespace creates a unique namespace for this environment that will
// not be automatically matched by the namespace selector for this test.
func (e *EndToEndEnvironment) WithUtilNamespace(ctx context.Context, fn func(ns *corev1.Namespace)) {
	require.NoError(e.t, endtoend.WithNamespace(ctx, e.Environment, e.unf, fn))
}

type EndToEndEnvironmentInstaller func(t *testing.T, ctx context.Context, e *EndToEndEnvironment)

func EndToEndEnvironmentWithTekton(t *testing.T, ctx context.Context, e *EndToEndEnvironment) {
	require.NoError(t, WithExclusive(ctx, "tekton", func() {
		InstallTektonPipeline(t, ctx, e.ControllerClient)
	}))
}

func EndToEndEnvironmentWithKnative(t *testing.T, ctx context.Context, e *EndToEndEnvironment) {
	require.NoError(t, WithExclusive(ctx, "knative-serving", func() {
		InstallKnativeServing(t, ctx, e.ControllerClient)
	}))
}

func EndToEndEnvironmentWithKourier(t *testing.T, ctx context.Context, e *EndToEndEnvironment) {
	require.NoError(t, WithExclusive(ctx, "kourier-system", func() {
		InstallKourier(t, ctx, e.ControllerClient, e.RESTMapper)
	}))
}

func EndToEndEnvironmentWithHostpathProvisioner(t *testing.T, ctx context.Context, e *EndToEndEnvironment) {
	require.NoError(t, WithExclusive(ctx, "hostpath", func() {
		InstallHostpathProvisioner(t, ctx, e.ControllerClient)
	}))
}

func EndToEndEnvironmentWithPVPool(t *testing.T, ctx context.Context, e *EndToEndEnvironment) {
	require.NoError(t, WithExclusive(ctx, "pvpool", func() {
		InstallPVPool(t, ctx, e.ControllerClient)
	}))
}

var _ EndToEndEnvironmentInstaller = EndToEndEnvironmentWithTekton
var _ EndToEndEnvironmentInstaller = EndToEndEnvironmentWithKnative
var _ EndToEndEnvironmentInstaller = EndToEndEnvironmentWithKourier
var _ EndToEndEnvironmentInstaller = EndToEndEnvironmentWithHostpathProvisioner
var _ EndToEndEnvironmentInstaller = EndToEndEnvironmentWithPVPool

func WithEndToEndEnvironment(t *testing.T, ctx context.Context, installers []EndToEndEnvironmentInstaller, fn func(e *EndToEndEnvironment)) {
	viper.SetEnvPrefix("relay_test_e2e")
	viper.AutomaticEnv()

	viper.SetDefault("disabled", false)
	viper.SetDefault("install_environment", true)

	// Don't inherit from $KUBECONFIG so people don't accidentally run these
	// tests against a cluster they care about.
	kubeconfigs := strings.TrimSpace(viper.GetString("kubeconfig"))
	switch {
	case viper.GetBool("disabled"):
		t.Skip("not running end-to-end tests because RELAY_TEST_E2E_DISABLED is set")
	case testing.Short():
		t.Skip("not running end-to-end tests with -short")
	case kubeconfigs == "":
		t.Skip("not running end-to-end tests without one or more Kubeconfigs specified by RELAY_TEST_E2E_KUBECONFIG")
	}

	opts := []endtoend.EnvironmentOption{
		endtoend.EnvironmentWithClientScheme(TestScheme),
		endtoend.EnvironmentWithClientKubeconfigs(filepath.SplitList(kubeconfigs)),
		endtoend.EnvironmentWithClientContext(viper.GetString("context")),
		endtoend.EnvironmentWithCRDDirectoryPaths{path.Join(ModuleDirectory, "manifests/resources")},
	}

	require.NoError(t, endtoend.WithEnvironment(opts, func(e *endtoend.Environment) {
		ls := map[string]string{
			"testutil.util.relay.sh/harness":   "end-to-end",
			"testutil.util.relay.sh/test.hash": testHash(t),
		}

		tkc, err := tekton.NewForConfig(e.RESTConfig)
		require.NoError(t, err, "failed to configure Tekton client")

		e2e := &EndToEndEnvironment{
			Environment:     e,
			Labels:          ls,
			LabelSelector:   metav1.LabelSelector{MatchLabels: ls},
			TektonInterface: tkc,
			t:               t,
			nf:              endtoend.NewTestNamespaceFactory(t, endtoend.NamespaceWithLabels(ls)),
			unf:             endtoend.NewTestNamespaceFactory(t, endtoend.NamespaceWithLabels(map[string]string{"testing.relay.sh/harness": "util"})),
		}

		if handler := viper.GetString("gvisor_handler"); handler != "" {
			require.NoError(t, WithExclusive(ctx, "gvisor", func() {
				InstallGVisorRuntimeClass(t, ctx, e.ControllerClient, handler)
			}))
			e2e.GVisorRuntimeClassName = "runsc"
		}

		if viper.GetBool("install_environment") {
			for _, installer := range installers {
				installer(t, ctx, e2e)
			}
		}

		fn(e2e)
	}))
}

func testHash(t *testing.T) string {
	h := sha256.Sum256([]byte(t.Name()))
	return hex.EncodeToString(h[:])[:63]
}
