package e2e_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/puppetlabs/leg/storage"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/middleware"
	"github.com/puppetlabs/relay-core/pkg/operator/admission"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/controller/tenant"
	"github.com/puppetlabs/relay-core/pkg/operator/controller/trigger"
	"github.com/puppetlabs/relay-core/pkg/operator/controller/workflow"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	"github.com/puppetlabs/relay-core/pkg/operator/obj"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Config struct {
	Namespace        *corev1.Namespace
	MetadataAPIURL   *url.URL
	Vault            *testutil.Vault
	Manager          manager.Manager
	ControllerConfig *config.WorkflowControllerConfig

	blobStore         storage.BlobStore
	dependencyManager *dependency.DependencyManager

	withVault                     bool
	withMetadataAPI               bool
	withMetadataAPIBoundInCluster bool
	withTenantReconciler          bool
	withWebhookTriggerReconciler  bool
	withWorkflowRunReconciler     bool
	withPodEnforcementAdmission   bool
	withVolumeClaimAdmission      bool
}

type ConfigOption func(cfg *Config)

func ConfigInNamespace(ns *corev1.Namespace) ConfigOption {
	return func(cfg *Config) {
		cfg.Namespace = ns
	}
}

func ConfigWithVault(cfg *Config) {
	cfg.withVault = true
}

func ConfigWithMetadataAPI(cfg *Config) {
	cfg.withMetadataAPI = true
}

func ConfigWithMetadataAPIBoundInCluster(cfg *Config) {
	ConfigWithMetadataAPI(cfg)
	cfg.withMetadataAPIBoundInCluster = true
}

func ConfigWithTenantReconciler(cfg *Config) {
	cfg.withTenantReconciler = true
}

func ConfigWithWebhookTriggerReconciler(cfg *Config) {
	ConfigWithTenantReconciler(cfg)
	cfg.withWebhookTriggerReconciler = true
}

func ConfigWithWorkflowRunReconciler(cfg *Config) {
	cfg.withWorkflowRunReconciler = true
}

func ConfigWithAllReconcilers(cfg *Config) {
	ConfigWithTenantReconciler(cfg)
	ConfigWithWebhookTriggerReconciler(cfg)
	ConfigWithWorkflowRunReconciler(cfg)
}

func ConfigWithPodEnforcementAdmission(cfg *Config) {
	cfg.withPodEnforcementAdmission = true
}

func ConfigWithVolumeClaimAdmission(cfg *Config) {
	cfg.withVolumeClaimAdmission = true
}

func ConfigWithEverything(cfg *Config) {
	ConfigWithMetadataAPI(cfg)
	ConfigWithAllReconcilers(cfg)
	ConfigWithPodEnforcementAdmission(cfg)
	ConfigWithVolumeClaimAdmission(cfg)
}

type doConfigFunc func(t *testing.T, cfg *Config, next func())

func doConfigNamespace(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if cfg.Namespace != nil {
			next()
			return
		}

		e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
			cfg.Namespace = ns
			cfg.ControllerConfig = &config.WorkflowControllerConfig{
				Namespace: ns.GetName(),
			}

			next()
		})
	}
}

func doConfigInit(t *testing.T, cfg *Config, next func()) {
	if !cfg.withWebhookTriggerReconciler && !cfg.withWorkflowRunReconciler {
		next()
		return
	}

	tmp, err := ioutil.TempDir("", "relay-e2e-")
	require.NoError(t, err)
	defer os.RemoveAll(tmp)

	blobStore, err := storage.NewBlobStore(url.URL{Scheme: "file", Path: tmp})
	require.NoError(t, err)

	cfg.blobStore = blobStore
	next()
}

func doConfigVault(t *testing.T, cfg *Config, next func()) {
	if !cfg.withVault && !cfg.withWebhookTriggerReconciler && !cfg.withWorkflowRunReconciler {
		next()
		return
	}

	testutil.WithVault(t, func(vcfg *testutil.Vault) {
		cfg.Vault = vcfg
		next()
	})
}

func doConfigMetadataAPI(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if !cfg.withMetadataAPI {
			next()
			return
		}

		log.Println("using metadata API")

		metadataAPI := httptest.NewServer(server.NewHandler(
			middleware.NewKubernetesAuthenticator(
				func(token string) (kubernetes.Interface, error) {
					rc := rest.AnonymousClientConfig(e2e.RESTConfig)
					rc.BearerToken = token

					return kubernetes.NewForConfig(rc)
				},
				middleware.KubernetesAuthenticatorWithKubernetesIntermediary(&authenticate.KubernetesInterface{
					Interface:       e2e.Interface,
					TektonInterface: e2e.TektonInterface,
				}),
				middleware.KubernetesAuthenticatorWithChainToVaultTransitIntermediary(cfg.Vault.Client, cfg.Vault.TransitPath, cfg.Vault.TransitKey),
				middleware.KubernetesAuthenticatorWithVaultResolver(cfg.Vault.Address, cfg.Vault.JWTAuthPath, cfg.Vault.JWTAuthRole),
			),
			server.WithTrustedProxyHops(1),
		))
		defer metadataAPI.Close()

		if !cfg.withMetadataAPIBoundInCluster {
			metadataAPIURL, err := url.Parse(metadataAPI.URL)
			require.NoError(t, err)

			cfg.MetadataAPIURL = metadataAPIURL
			next()
		} else {
			e2e.WithUtilNamespace(t, ctx, func(ns *corev1.Namespace) {
				testutil.WithServiceBoundToHostHTTP(t, ctx, e2e.RESTConfig, e2e.Interface, metadataAPI.URL, metav1.ObjectMeta{Namespace: ns.GetName()}, func(caPEM []byte, svc *corev1.Service) {
					cfg.MetadataAPIURL = &url.URL{
						Scheme: "http",
						Host:   fmt.Sprintf("%s.%s", svc.GetName(), svc.GetNamespace()),
					}
					next()
				})
			})
		}
	}
}

func doConfigDependencyManager(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if !cfg.withWebhookTriggerReconciler && !cfg.withWorkflowRunReconciler {
			next()
			return
		}

		require.NotNil(t, cfg.Namespace)
		require.NotNil(t, cfg.Vault)
		require.NotNil(t, cfg.blobStore)

		imagePullSecret := obj.NewImagePullSecret(client.ObjectKey{
			Namespace: cfg.Namespace.GetName(),
			Name:      "docker-registry-system",
		})
		imagePullSecret.Object.Data = map[string][]byte{
			".dockerconfigjson": []byte(`{}`),
		}
		require.NoError(t, imagePullSecret.Persist(ctx, e2e.ControllerRuntimeClient))

		wcc := &config.WorkflowControllerConfig{
			Namespace:               cfg.Namespace.GetName(),
			ImagePullSecret:         imagePullSecret.Key.Name,
			MaxConcurrentReconciles: 16,
			MetadataAPIURL:          cfg.MetadataAPIURL,
			VaultTransitPath:        cfg.Vault.TransitPath,
			VaultTransitKey:         cfg.Vault.TransitKey,
		}

		deps, err := dependency.NewDependencyManager(wcc, e2e.RESTConfig, cfg.Vault.Client, cfg.Vault.JWTSigner, cfg.blobStore, metrics)
		require.NoError(t, err)

		cfg.dependencyManager = deps
		cfg.Manager = deps.Manager
		cfg.ControllerConfig = deps.Config
		next()
	}
}

func doConfigReconcilers(t *testing.T, cfg *Config, next func()) {
	if cfg.withTenantReconciler {
		log.Println("using tenant reconciler")

		require.NotNil(t, cfg.Namespace)
		require.NotNil(t, cfg.Manager)

		require.NoError(t, tenant.Add(cfg.Manager, cfg.ControllerConfig))
	}

	if cfg.withWebhookTriggerReconciler {
		log.Println("using webhook trigger reconciler")

		require.NotNil(t, cfg.dependencyManager)

		require.NoError(t, trigger.Add(cfg.dependencyManager))
	}

	if cfg.withWorkflowRunReconciler {
		log.Println("using workflow run reconciler")

		require.NotNil(t, cfg.dependencyManager)

		require.NoError(t, workflow.Add(cfg.dependencyManager))
	}

	next()
}

func doConfigPodEnforcementAdmission(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if !cfg.withPodEnforcementAdmission {
			next()
			return
		}

		opts := []admission.PodEnforcementHandlerOption{
			admission.PodEnforcementHandlerWithStandaloneMode(true),
			admission.PodEnforcementHandlerWithRuntimeClassName(e2e.GVisorRuntimeClassName),
		}
		testutil.WithPodEnforcementAdmissionRegistration(t, ctx, e2e, cfg.Manager, opts, nil, next)
	}
}

func doConfigVolumeClaimAdmission(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if !cfg.withVolumeClaimAdmission {
			next()
			return
		}

		testutil.WithVolumeClaimAdmissionRegistration(t, ctx, e2e, cfg.Manager, nil, nil, next)
	}
}

func doConfigLifecycle(t *testing.T, cfg *Config, next func()) {
	var wg sync.WaitGroup

	ch := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		require.NoError(t, cfg.Manager.Start(ch))
	}()
	defer func() {
		close(ch)
		wg.Wait()
	}()

	next()
}

func doConfigUser(fn func(cfg *Config)) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		fn(cfg)
		next()
	}
}

func doConfigCleanup(t *testing.T, cfg *Config, next func()) {
	defer next()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NotNil(t, cfg.Namespace)

	var del []runtime.Object

	tl := &relayv1beta1.TenantList{}
	require.NoError(t, e2e.ControllerRuntimeClient.List(ctx, tl, client.InNamespace(cfg.Namespace.GetName())))
	if len(tl.Items) > 0 {
		log.Printf("removing %d stale tenant(s)", len(tl.Items))
		for _, t := range tl.Items {
			func(t relayv1beta1.Tenant) {
				del = append(del, &t)
			}(t)
		}
	}

	wtl := &relayv1beta1.WebhookTriggerList{}
	require.NoError(t, e2e.ControllerRuntimeClient.List(ctx, wtl, client.InNamespace(cfg.Namespace.GetName())))
	if len(wtl.Items) > 0 {
		log.Printf("removing %d stale webhook trigger(s)", len(wtl.Items))
		for _, wt := range wtl.Items {
			func(wt relayv1beta1.WebhookTrigger) {
				del = append(del, &wt)
			}(wt)
		}
	}

	wrl := &nebulav1.WorkflowRunList{}
	require.NoError(t, e2e.ControllerRuntimeClient.List(ctx, wrl, client.InNamespace(cfg.Namespace.GetName())))
	if len(wrl.Items) > 0 {
		log.Printf("removing %d stale workflow run(s)", len(wrl.Items))
		for _, wr := range wrl.Items {
			func(wr nebulav1.WorkflowRun) {
				del = append(del, &wr)
			}(wr)
		}
	}

	for _, obj := range del {
		assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, obj))
	}

	for _, obj := range del {
		assert.NoError(t, testutil.WaitForObjectDeletion(ctx, e2e.ControllerRuntimeClient, obj))
	}
}

func WithConfig(t *testing.T, ctx context.Context, opts []ConfigOption, fn func(cfg *Config)) {
	mgr, err := ctrl.NewManager(e2e.RESTConfig, ctrl.Options{
		Scheme:             testutil.TestScheme,
		MetricsBindAddress: "0",
	})
	require.NoError(t, err)

	cfg := &Config{
		MetadataAPIURL: &url.URL{Scheme: "http", Host: "stub.example.com"},
		Manager:        mgr,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	chain := []doConfigFunc{
		doConfigNamespace(ctx),
		doConfigInit,
		doConfigVault,
		doConfigMetadataAPI(ctx),
		doConfigDependencyManager(ctx),
		doConfigReconcilers,
		doConfigPodEnforcementAdmission(ctx),
		doConfigVolumeClaimAdmission(ctx),
		doConfigLifecycle,
		doConfigUser(fn),
		doConfigCleanup,
	}

	// Execute chain.
	i := -1
	var run func()
	run = func() {
		i++
		if i < len(chain) {
			chain[i](t, cfg, run)
		}
	}
	run()

	require.Equal(t, len(chain), i, "config chain did not run to completion")
}
