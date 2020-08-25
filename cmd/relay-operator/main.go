package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"io/ioutil"
	"log"
	"net/url"
	"os"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/delegates"
	metricsserver "github.com/puppetlabs/horsehead/v2/instrumentation/metrics/server"
	"github.com/puppetlabs/horsehead/v2/storage"
	_ "github.com/puppetlabs/horsehead/v2/storage/file"
	_ "github.com/puppetlabs/horsehead/v2/storage/gcs"
	"github.com/puppetlabs/relay-core/pkg/admission"
	"github.com/puppetlabs/relay-core/pkg/config"
	"github.com/puppetlabs/relay-core/pkg/controller/tenant"
	"github.com/puppetlabs/relay-core/pkg/controller/trigger"
	"github.com/puppetlabs/relay-core/pkg/controller/workflow"
	"github.com/puppetlabs/relay-core/pkg/dependency"
	jose "gopkg.in/square/go-jose.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func main() {
	// We use a custom flag set because Tekton's API has the default flag set
	// configured too, which makes our command help make no sense.
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	environment := fs.String("environment", "dev", "the environment this operator is running in")
	standalone := fs.Bool("standalone", false, "enables standalone mode which runs workflows without multi-tenant network security protections")
	kubeconfig := fs.String("kubeconfig", "", "path to kubeconfig file. Only required if running outside of a cluster.")
	kubeMasterURL := fs.String("kube-master-url", "", "url to the kubernetes master")
	kubeNamespace := fs.String("kube-namespace", "", "an optional working namespace to restrict to for watching CRDs")
	imagePullSecret := fs.String("image-pull-secret", "", "the optionally namespaced name of the image pull secret to use for system images")
	storageAddr := fs.String("storage-addr", "", "the storage URL to upload logs into")
	numWorkers := fs.Int("num-workers", 2, "the number of worker threads to spawn that process Workflow resources")
	metricsEnabled := fs.Bool("metrics-enabled", false, "enables the metrics collection and server")
	metricsServerBindAddr := fs.String("metrics-server-bind-addr", "localhost:3050", "the host:port to bind the metrics server to")
	jwtSigningKeyFile := fs.String("jwt-signing-key-file", "", "path to a PEM-encoded RSA JWT key to use for signing step tokens")
	vaultTransitPath := fs.String("vault-transit-path", "transit", "path to the Vault secrets engine to use for encrypting step tokens")
	vaultTransitKey := fs.String("vault-transit-key", "metadata-api", "the Vault transit key to use")
	metadataAPIURLStr := fs.String("metadata-api-url", "", "URL to the metadata API")
	webhookServerPort := fs.Int("webhook-server-port", 443, "the port to listen on for webhook requests")
	webhookServerKeyDir := fs.String("webhook-server-key-dir", "", "path to a directory containing two files, tls.key and tls.crt, to secure the webhook server")
	tenantSandboxing := fs.Bool("tenant-sandboxing", false, "enables gVisor sandbox for tenant pods")
	sentryDSN := fs.String("sentry-dsn", "", "the Sentry DSN to use for error reporting")
	dynamicRBACBinding := fs.Bool("dynamic-rbac-binding", false, "enable if RBAC rules are set up dynamically for the operator to reduce unhelpful reported errors")
	toolInjectionImage := fs.String("tool-injection-image", "relaysh/relay-runtime-tools", "image to use for the tool injection suite")

	fs.Parse(os.Args[1:])

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	storageUrl, err := url.Parse(*storageAddr)
	if err != nil {
		log.Fatal("Error parsing the -storage-addr", err)
	}

	metadataAPIURL, err := url.Parse(*metadataAPIURLStr)
	if err != nil {
		log.Fatal("Error parsing -metadata-api-url", err)
	}

	blobStore, err := storage.NewBlobStore(*storageUrl)
	if err != nil {
		log.Fatal("Error initializing the storage client from the -storage-addr", err)
	}

	if *webhookServerKeyDir == "" {
		log.Fatal("The webhook server key directory -webhook-server-key-dir must be specified")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	metricsOpts := metrics.Options{
		DelegateType:  delegates.PrometheusDelegate,
		ErrorBehavior: metrics.ErrorBehaviorLog,
	}

	mets, err := metrics.NewNamespace("workflow_controller", metricsOpts)
	if err != nil {
		log.Fatal("Error setting up metrics server")
	}

	if *metricsEnabled {
		ms := metricsserver.New(mets, metricsserver.Options{
			BindAddr: *metricsServerBindAddr,
		})

		go ms.Run(ctx)
	}

	kcfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: *kubeconfig},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: *kubeMasterURL}},
	)

	kcc, err := kcfg.ClientConfig()
	if err != nil {
		log.Fatal("Error creating kubernetes config", err)
	}

	vc, err := vaultapi.NewClient(vaultapi.DefaultConfig())
	if err != nil {
		log.Fatal("Error creating Vault client", err)
	}

	jwtSigningKeyBytes, err := ioutil.ReadFile(*jwtSigningKeyFile)
	if err != nil {
		log.Fatal("Error reading JWT signing key file", err)
	}

	jwtSigningKeyBlock, _ := pem.Decode(jwtSigningKeyBytes)
	if jwtSigningKeyBlock == nil {
		log.Fatal("Error parsing PEM")
	} else if jwtSigningKeyBlock.Type != "RSA PRIVATE KEY" {
		log.Fatal("PEM file does not contain an RSA private key")
	}

	jwtSigningKey, err := x509.ParsePKCS1PrivateKey(jwtSigningKeyBlock.Bytes)
	if err != nil {
		log.Fatal("Error parsing RSA private key", err)
	}

	jwtSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: jwtSigningKey}, &jose.SignerOptions{})
	if err != nil {
		log.Fatal("Error creating signer for JWTs", err)
	}

	var alertsDelegate alerts.DelegateFunc
	if *sentryDSN != "" {
		var err error
		alertsDelegate, err = alerts.DelegateToSentry(*sentryDSN)
		if err != nil {
			log.Fatal("Error initializing Sentry", err)
		}
	}

	cfg := &config.WorkflowControllerConfig{
		Environment:             *environment,
		Standalone:              *standalone,
		Namespace:               *kubeNamespace,
		ImagePullSecret:         *imagePullSecret,
		MaxConcurrentReconciles: *numWorkers,
		MetadataAPIURL:          metadataAPIURL,
		VaultTransitPath:        *vaultTransitPath,
		VaultTransitKey:         *vaultTransitKey,
		WebhookServerPort:       *webhookServerPort,
		WebhookServerKeyDir:     *webhookServerKeyDir,
		AlertsDelegate:          alertsDelegate,
		DynamicRBACBinding:      *dynamicRBACBinding,
		ToolInjectionImage:      *toolInjectionImage,
	}

	dm, err := dependency.NewDependencyManager(cfg, kcc, vc, jwtSigner, blobStore, mets)
	if err != nil {
		log.Fatal("Error creating controller dependency builder", err)
	}

	if err := workflow.Add(dm); err != nil {
		log.Fatal("Could not add all controllers to operator manager", err)
	}

	if err := tenant.Add(dm.Manager, cfg); err != nil {
		log.Fatal("Could not add all controllers to operator manager", err)
	}

	if err := trigger.Add(dm); err != nil {
		log.Fatal("Could not add all controllers to operator manager", err)
	}

	dm.Manager.GetWebhookServer().Register("/mutate/pod-enforcement", &webhook.Admission{
		Handler: admission.NewPodEnforcementHandler(
			admission.PodEnforcementHandlerWithSandboxing(*tenantSandboxing),
			admission.PodEnforcementHandlerWithStandaloneMode(*standalone),
		),
	})

	dm.Manager.GetWebhookServer().Register("/mutate/volume-claim", &webhook.Admission{
		Handler: admission.NewVolumeClaimHandler(),
	})

	if err := dm.Manager.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal("Manager exited non-zero", err)
	}
}
