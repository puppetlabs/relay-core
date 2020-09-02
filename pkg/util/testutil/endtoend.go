package testutil

import (
	"context"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/util/retry"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

type EndToEndEnvironment struct {
	RESTConfig              *rest.Config
	RESTMapper              meta.RESTMapper
	ControllerRuntimeClient client.Client
	Interface               kubernetes.Interface
	TektonInterface         tekton.Interface
}

func (e *EndToEndEnvironment) WithTestNamespace(t *testing.T, ctx context.Context, fn func(ns *corev1.Namespace)) {
	namePrefix := strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r | 0x20
		} else if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') {
			return r
		}

		return '-'
	}, t.Name())
	if len(namePrefix) > 28 {
		// Leave some room for names added to the end within tests.
		namePrefix = namePrefix[:28]
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("relay-e2e-%s-", namePrefix),
			Labels: map[string]string{
				"testing.relay.sh/harness":    "end-to-end",
				"testing.relay.sh/tools-volume-claim": "true",
			},
		},
	}
	require.NoError(t, e.ControllerRuntimeClient.Create(ctx, ns))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		assert.NoError(t, e.ControllerRuntimeClient.Delete(ctx, ns))
	}()

	// Wait for default service account to be populated.
	require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
		sa := &corev1.ServiceAccount{}
		if err := e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Namespace: ns.GetName(), Name: "default"}, sa); errors.IsNotFound(err) {
			return retry.RetryTransient(fmt.Errorf("waiting for service account"))
		} else if err != nil {
			return retry.RetryPermanent(err)
		}

		return retry.RetryPermanent(nil)
	}))

	fn(ns)
}

type EndToEndEnvironmentOption func(ctx context.Context, e *EndToEndEnvironment) error

func EndToEndEnvironmentWithTekton(ctx context.Context, e *EndToEndEnvironment) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return doInstallTektonPipeline(ctx, e.ControllerRuntimeClient)
}

func EndToEndEnvironmentWithKnative(ctx context.Context, e *EndToEndEnvironment) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return doInstallKnativeServing(ctx, e.ControllerRuntimeClient)
}

func EndToEndEnvironmentWithAmbassador(ctx context.Context, e *EndToEndEnvironment) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return doInstallAmbassador(ctx, e.ControllerRuntimeClient, e.RESTMapper)
}

func EndToEndEnvironmentWithHostpathProvisioner(ctx context.Context, e *EndToEndEnvironment) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	return doInstallHostpathProvisioner(ctx, e.ControllerRuntimeClient)
}

var _ EndToEndEnvironmentOption = EndToEndEnvironmentWithTekton
var _ EndToEndEnvironmentOption = EndToEndEnvironmentWithKnative
var _ EndToEndEnvironmentOption = EndToEndEnvironmentWithAmbassador
var _ EndToEndEnvironmentOption = EndToEndEnvironmentWithHostpathProvisioner

func doEndToEndEnvironment(fn func(e *EndToEndEnvironment), opts ...EndToEndEnvironmentOption) (bool, error) {
	// We'll allow 30 minutes to attach the environment and run the test. This
	// should give us time to acquire the exclusive lock but puts a cap on any
	// underlying issues causing timeouts in CI.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	viper.SetEnvPrefix("relay_test_e2e")
	viper.AutomaticEnv()

	viper.SetDefault("disabled", false)
	viper.SetDefault("install_environment", true)

	if viper.GetBool("disabled") {
		return false, nil
	}

	// Don't inherit from $KUBECONFIG so people don't accidentally run these
	// tests against a cluster they care about.
	kubeconfigs := strings.TrimSpace(viper.GetString("kubeconfig"))
	if kubeconfigs == "" {
		return true, fmt.Errorf("end-to-end tests require the RELAY_TEST_E2E_KUBECONFIG environment variable to be set to the path of a valid Kubeconfig")
	}

	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			Precedence:       filepath.SplitList(kubeconfigs),
			WarnIfAllMissing: true,
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: viper.GetString("context"),
		},
	).ClientConfig()
	if err != nil {
		return true, fmt.Errorf("failed to create Kubernetes cluster configuration: %+v", err)
	}

	env := &envtest.Environment{
		Config: cfg,
		CRDDirectoryPaths: []string{
			path.Join(ModuleDirectory, "manifests/resources"),
		},
		AttachControlPlaneOutput: true,
		UseExistingCluster:       func(b bool) *bool { return &b }(true),
	}

	// End-to-end tests require an exclusive lock on the environment so, e.g.,
	// we don't try to simultaneously install Tekton in two different packages.
	release, err := Exclusive(ctx, LockEndToEndEnvironment)
	if err != nil {
		return true, fmt.Errorf("failed to acquire exclusive lock: %+v", err)
	}
	defer release()

	cfg, err = env.Start()
	if err != nil {
		return true, fmt.Errorf("failed to connect to Kubernetes cluster: %+v", err)
	}
	defer env.Stop()

	log.Println("connected to Kubernetes cluster", cfg.Host)

	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		return true, fmt.Errorf("failed to configure client resource discovery: %+v", err)
	}

	client, err := client.New(cfg, client.Options{
		Scheme: TestScheme,
		Mapper: mapper,
	})
	if err != nil {
		return true, fmt.Errorf("failed to configure client: %+v", err)
	}

	ifc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return true, fmt.Errorf("failed to configure client: %+v", err)
	}

	tkc, err := tekton.NewForConfig(cfg)
	if err != nil {
		return true, fmt.Errorf("failed to configure Tekton client: %+v", err)
	}

	e := &EndToEndEnvironment{
		RESTConfig:              cfg,
		RESTMapper:              mapper,
		ControllerRuntimeClient: client,
		Interface:               ifc,
		TektonInterface:         tkc,
	}

	if viper.GetBool("install_environment") {
		for _, opt := range opts {
			if err := opt(ctx, e); err != nil {
				return true, err
			}
		}
	}

	fn(e)
	return true, nil
}

func WithEndToEndEnvironment(t *testing.T, fn func(e *EndToEndEnvironment), opts ...EndToEndEnvironmentOption) {
	enabled, err := doEndToEndEnvironment(fn, opts...)
	require.NoError(t, err)
	if !enabled {
		t.Log("end-to-end tests disabled by configuration")
	}
}

func RunEndToEnd(m *testing.M, fn func(e *EndToEndEnvironment), opts ...EndToEndEnvironmentOption) int {
	if enabled, err := doEndToEndEnvironment(fn, opts...); err != nil {
		log.Println(err)
		return 1
	} else if !enabled {
		log.Println("end-to-end tests disabled by configuration")
		return 0
	}

	return m.Run()
}
