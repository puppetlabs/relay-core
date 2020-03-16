package main

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"

	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/delegates"
	metricsserver "github.com/puppetlabs/horsehead/v2/instrumentation/metrics/server"
	"github.com/puppetlabs/horsehead/v2/storage"
	_ "github.com/puppetlabs/nebula-libs/storage/file/v2"
	_ "github.com/puppetlabs/nebula-libs/storage/gcs/v2"
	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/workflow"
	"github.com/puppetlabs/nebula-tasks/pkg/dependency"
	"github.com/puppetlabs/nebula-tasks/pkg/secrets/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/util"
	tekv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	// We use a custom flag set because Tekton's API has the default flag set
	// configured too, which makes our command help make no sense.
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	kubeconfig := fs.String("kubeconfig", "", "path to kubeconfig file. Only required if running outside of a cluster.")
	kubeMasterURL := fs.String("kube-master-url", "", "url to the kubernetes master")
	kubeNamespace := fs.String("kube-namespace", "", "an optional working namespace to run the controller as. Only required if running outside of a cluster.")
	vaultAddr := fs.String("vault-addr", "http://localhost:8200", "address to the vault server")
	vaultToken := fs.String("vault-token", "", "token used to authenticate with the vault server")
	vaultEngineMount := fs.String("vault-engine-mount", "nebula", "the engine mount to craft paths from")
	storageAddr := fs.String("storage-addr", "", "the storage URL to upload logs into")
	metadataServiceImage := fs.String("metadata-service-image", "gcr.io/nebula-235818/nebula-metadata-api:latest", "the image and tag to use for the metadata service api")
	metadataServiceImagePullSecret := fs.String("metadata-service-image-pull-secret", "", "the optionally namespaced name of the image pull secret to use for the metadata service")
	metadataServiceVaultAddr := fs.String("metadata-service-vault-addr", "", "the address to use when authenticating the metadata service to Vault")
	metadataServiceVaultAuthMountPath := fs.String("metadata-service-vault-auth-mount-path", "", "the mount path to use when authenticating the metadata service to Vault")
	metadataServiceCheckEnabled := fs.Bool("metadata-service-check-enabled", true, "whether to enable checking the metadata service over HTTP")
	numWorkers := fs.Int("num-workers", 2, "the number of worker threads to spawn that process Workflow resources")
	metricsEnabled := fs.Bool("metrics-enabled", false, "enables the metrics collection and server")
	metricsServerBindAddr := fs.String("metrics-server-bind-addr", "localhost:3050", "the host:port to bind the metrics server to")
	whenConditionsImage := fs.String("when-conditions-image", "gcr.io/nebula-235818/nebula-conditions:latest", "the image and tag to use for evaluating when conditions")

	fs.Parse(os.Args[1:])

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	vc, err := vault.NewVaultAuth(&vault.Config{
		Addr:             *vaultAddr,
		K8sAuthMountPath: *metadataServiceVaultAuthMountPath,
		Token:            *vaultToken,
		EngineMount:      *vaultEngineMount,
	})
	if err != nil {
		log.Fatal("Error initializing the vault client from the -vault-addr -vault-token and -vault-engine-mount", err)
	}

	storageUrl, err := url.Parse(*storageAddr)
	if err != nil {
		log.Fatal("Error parsing the -storage-addr", err)
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

	namespace, err := util.LookupNamespace(*kubeNamespace)
	if err != nil {
		log.Fatal("Error looking up namespace")
	}

	cfg := &config.WorkflowControllerConfig{
		Namespace:                         namespace,
		MetadataServiceImage:              *metadataServiceImage,
		MetadataServiceImagePullSecret:    *metadataServiceImagePullSecret,
		MetadataServiceVaultAddr:          *metadataServiceVaultAddr,
		MetadataServiceVaultAuthMountPath: *metadataServiceVaultAuthMountPath,
		MetadataServiceCheckEnabled:       *metadataServiceCheckEnabled,
		WhenConditionsImage:               *whenConditionsImage,
		MaxConcurrentReconciles:           *numWorkers,
	}

	dm, err := dependency.NewDependencyManager(kcc, cfg, vc, blobStore, mets)
	if err != nil {
		log.Fatal("Error creating controller dependency builder", err)
	}

	config := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(config, ctrl.Options{})
	if err != nil {
		log.Fatal("Unable to create new manager", err)
	}

	if err := nebulav1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal("Could not add manager scheme", err)
	}

	if err := tekv1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal("Could not add manager scheme", err)
	}

	if err := workflow.Add(mgr, dm); err != nil {
		log.Fatal("Could not add all controllers to operator manager", err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatal("Manager exited non-zero", err)
	}
}
