package e2e_test

import (
	"context"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/puppetlabs/leg/k8sutil/pkg/app/tunnel"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/storage"
	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	pvpoolv1alpha1obj "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1/obj"
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
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Config struct {
	Environment           *testutil.EndToEndEnvironment
	Namespace             *corev1.Namespace
	MetadataAPIURL        *url.URL
	Vault                 *testutil.Vault
	Manager               manager.Manager
	ToolInjectionPoolName string
	ControllerConfig      *config.WorkflowControllerConfig

	blobStore         storage.BlobStore
	dependencyManager *dependency.DependencyManager

	withoutCleanup                bool
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

func ConfigWithToolInjectionPoolName(name string) ConfigOption {
	return func(cfg *Config) {
		cfg.ToolInjectionPoolName = name
	}
}

func ConfigWithoutCleanup(cfg *Config) {
	cfg.withoutCleanup = true
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

		cfg.Environment.WithTestNamespace(ctx, func(ns *corev1.Namespace) {
			cfg.Namespace = ns
			cfg.ControllerConfig = &config.WorkflowControllerConfig{
				Namespace: ns.GetName(),
			}

			next()
		})
	}
}

func doConfigInit(t *testing.T, cfg *Config, next func()) {
	if !cfg.withTenantReconciler && !cfg.withWebhookTriggerReconciler && !cfg.withWorkflowRunReconciler {
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
	if !cfg.withVault && !cfg.withTenantReconciler && !cfg.withWebhookTriggerReconciler && !cfg.withWorkflowRunReconciler {
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
					rc := rest.AnonymousClientConfig(cfg.Environment.RESTConfig)
					rc.BearerToken = token

					return kubernetes.NewForConfig(rc)
				},
				middleware.KubernetesAuthenticatorWithKubernetesIntermediary(&authenticate.KubernetesInterface{
					Interface:       cfg.Environment.StaticClient,
					TektonInterface: cfg.Environment.TektonInterface,
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
			cfg.Environment.WithUtilNamespace(ctx, func(ns *corev1.Namespace) {
				tun, err := tunnel.ApplyHTTP(ctx, cfg.Environment.ControllerClient, client.ObjectKey{Namespace: ns.GetName(), Name: "tunnel"})
				require.NoError(t, err)

				require.NoError(t, tunnel.WithHTTPConnection(ctx, cfg.Environment.RESTConfig, tun, metadataAPI.URL, func(ctx context.Context) {
					cfg.MetadataAPIURL = &url.URL{
						Scheme: "http",
						Host:   tun.Service.DNSName(),
					}
					next()
				}))
			})
		}
	}
}

func doConfigDependencyManager(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if !cfg.withTenantReconciler && !cfg.withWebhookTriggerReconciler && !cfg.withWorkflowRunReconciler {
			next()
			return
		}

		require.NotNil(t, cfg.Namespace)
		require.NotNil(t, cfg.Vault)
		require.NotNil(t, cfg.blobStore)

		imagePullSecret := corev1obj.NewImagePullSecret(client.ObjectKey{
			Namespace: cfg.Namespace.GetName(),
			Name:      "docker-registry-system",
		})
		_, err := imagePullSecret.Load(ctx, cfg.Environment.ControllerClient)
		require.NoError(t, err)
		imagePullSecret.Object.Data = map[string][]byte{
			".dockerconfigjson": []byte(`{}`),
		}
		require.NoError(t, imagePullSecret.Persist(ctx, cfg.Environment.ControllerClient))

		wcc := &config.WorkflowControllerConfig{
			Standalone:              true,
			Namespace:               cfg.Namespace.GetName(),
			ImagePullSecret:         imagePullSecret.Key.Name,
			MaxConcurrentReconciles: 16,
			MetadataAPIURL:          cfg.MetadataAPIURL,
			VaultTransitPath:        cfg.Vault.TransitPath,
			VaultTransitKey:         cfg.Vault.TransitKey,
		}

		if cfg.withTenantReconciler {
			pool := pvpoolv1alpha1obj.NewPool(client.ObjectKey{
				Namespace: cfg.Namespace.GetName(),
				Name:      cfg.ToolInjectionPoolName,
			})
			_, err := pool.Load(ctx, cfg.Environment.ControllerClient)
			require.NoError(t, err)
			pool.Object.Spec = pvpoolv1alpha1.PoolSpec{
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"testing.relay.sh/selector": cfg.ToolInjectionPoolName,
					},
				},
				Template: pvpoolv1alpha1.PersistentVolumeClaimTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"testing.relay.sh/selector": cfg.ToolInjectionPoolName,
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
							corev1.ReadOnlyMany,
						},
						StorageClassName: pointer.StringPtr("relay-hostpath"),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("50Mi"),
							},
						},
					},
				},
				InitJob: &pvpoolv1alpha1.MountJob{
					Template: pvpoolv1alpha1.JobTemplate{
						Spec: batchv1.JobSpec{
							BackoffLimit:          pointer.Int32Ptr(2),
							ActiveDeadlineSeconds: pointer.Int64Ptr(60),
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name: "init",
											// XXX: TODO: This should come from ko!
											Image:           "relaysh/relay-runtime-tools:latest",
											ImagePullPolicy: corev1.PullAlways,
											Command:         []string{"cp"},
											Args:            []string{"-r", "/relay/runtime/tools/.", "/workspace"},
											VolumeMounts: []corev1.VolumeMount{
												{Name: "workspace", MountPath: "/workspace"},
											},
										},
									},
								},
							},
						},
					},
				},
			}
			require.NoError(t, pool.Persist(ctx, cfg.Environment.ControllerClient))

			wcc.ToolInjectionPool = pool.Key
		}

		deps, err := dependency.NewDependencyManager(wcc, cfg.Environment.RESTConfig, cfg.Vault.Client, cfg.Vault.JWTSigner, cfg.blobStore, metrics)
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
			admission.PodEnforcementHandlerWithRuntimeClassName(cfg.Environment.GVisorRuntimeClassName),
		}
		testutil.WithPodEnforcementAdmissionRegistration(t, ctx, cfg.Environment, cfg.Manager, opts, nil, next)
	}
}

func doConfigVolumeClaimAdmission(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		if !cfg.withVolumeClaimAdmission {
			next()
			return
		}

		testutil.WithVolumeClaimAdmissionRegistration(t, ctx, cfg.Environment, cfg.Manager, nil, nil, next)
	}
}

func doConfigLifecycle(ctx context.Context) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		var wg sync.WaitGroup

		ctx, cancel := context.WithCancel(ctx)

		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, cfg.Manager.Start(ctx))
		}()
		defer func() {
			cancel()
			wg.Wait()
		}()

		next()
	}
}

func doConfigUser(fn func(cfg *Config)) doConfigFunc {
	return func(t *testing.T, cfg *Config, next func()) {
		defer next()
		fn(cfg)
	}
}

func doConfigCleanup(t *testing.T, cfg *Config, next func()) {
	defer next()

	if cfg.withoutCleanup {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NotNil(t, cfg.Namespace)

	var del []client.Object

	tl := &relayv1beta1.TenantList{}
	require.NoError(t, cfg.Environment.ControllerClient.List(ctx, tl, client.InNamespace(cfg.Namespace.GetName())))
	if len(tl.Items) > 0 {
		log.Printf("removing %d stale tenant(s)", len(tl.Items))
		for _, t := range tl.Items {
			func(t relayv1beta1.Tenant) {
				del = append(del, &t)
			}(t)
		}
	}

	wtl := &relayv1beta1.WebhookTriggerList{}
	require.NoError(t, cfg.Environment.ControllerClient.List(ctx, wtl, client.InNamespace(cfg.Namespace.GetName())))
	if len(wtl.Items) > 0 {
		log.Printf("removing %d stale webhook trigger(s)", len(wtl.Items))
		for _, wt := range wtl.Items {
			func(wt relayv1beta1.WebhookTrigger) {
				del = append(del, &wt)
			}(wt)
		}
	}

	wrl := &nebulav1.WorkflowRunList{}
	require.NoError(t, cfg.Environment.ControllerClient.List(ctx, wrl, client.InNamespace(cfg.Namespace.GetName())))
	if len(wrl.Items) > 0 {
		log.Printf("removing %d stale workflow run(s)", len(wrl.Items))
		for _, wr := range wrl.Items {
			func(wr nebulav1.WorkflowRun) {
				del = append(del, &wr)
			}(wr)
		}
	}

	for _, obj := range del {
		assert.NoError(t, cfg.Environment.ControllerClient.Delete(ctx, obj))
	}

	for _, obj := range del {
		assert.NoError(t, testutil.WaitForObjectDeletion(ctx, cfg.Environment.ControllerClient, obj))
	}
}

func WithConfig(t *testing.T, ctx context.Context, opts []ConfigOption, fn func(cfg *Config)) {
	cfg := &Config{
		MetadataAPIURL:        &url.URL{Scheme: "http", Host: "stub.example.com"},
		ToolInjectionPoolName: "tool-injection-pool",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var installers []testutil.EndToEndEnvironmentInstaller
	if cfg.withWorkflowRunReconciler {
		installers = append(installers, testutil.EndToEndEnvironmentWithTekton)
	}
	if cfg.withWebhookTriggerReconciler {
		installers = append(installers, testutil.EndToEndEnvironmentWithAmbassador)
		installers = append(installers, testutil.EndToEndEnvironmentWithKnative)
	}
	if cfg.withTenantReconciler || cfg.withVolumeClaimAdmission {
		installers = append(installers, testutil.EndToEndEnvironmentWithHostpathProvisioner)
		installers = append(installers, testutil.EndToEndEnvironmentWithPVPool)
	}

	testutil.WithEndToEndEnvironment(
		t,
		ctx,
		installers,
		func(e2e *testutil.EndToEndEnvironment) {
			mgr, err := ctrl.NewManager(e2e.RESTConfig, ctrl.Options{
				Scheme:             testutil.TestScheme,
				MetricsBindAddress: "0",
			})
			require.NoError(t, err)

			cfg.Environment = e2e
			cfg.Manager = mgr

			chain := []doConfigFunc{
				doConfigNamespace(ctx),
				doConfigInit,
				doConfigVault,
				doConfigMetadataAPI(ctx),
				doConfigDependencyManager(ctx),
				doConfigReconcilers,
				doConfigPodEnforcementAdmission(ctx),
				doConfigVolumeClaimAdmission(ctx),
				doConfigLifecycle(ctx),
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
		},
	)
}
