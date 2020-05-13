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
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/delegates"
	metricsserver "github.com/puppetlabs/horsehead/v2/instrumentation/metrics/server"
	"github.com/puppetlabs/horsehead/v2/storage"
	_ "github.com/puppetlabs/nebula-libs/storage/file/v2"
	_ "github.com/puppetlabs/nebula-libs/storage/gcs/v2"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/workflow"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	// We use a custom flag set because Tekton's API has the default flag set
	// configured too, which makes our command help make no sense.
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

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

	cfg := &config.WorkflowControllerConfig{
		Namespace:               *kubeNamespace,
		ImagePullSecret:         *imagePullSecret,
		MaxConcurrentReconciles: *numWorkers,
		MetadataAPIURL:          metadataAPIURL,
		VaultTransitPath:        *vaultTransitPath,
		VaultTransitKey:         *vaultTransitKey,
	}

	dm, err := dependency.NewDependencyManager(cfg, kcc, vc, jwtSigner, blobStore, mets)
	if err != nil {
		log.Fatal("Error creating controller dependency builder", err)
	}

	if err := workflow.Add(dm); err != nil {
		log.Fatal("Could not add all controllers to operator manager", err)
	}

	if err := dm.Manager.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal("Manager exited non-zero", err)
	}
}
