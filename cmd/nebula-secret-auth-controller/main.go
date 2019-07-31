package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/storage/gcs"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controllers/secretauth"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets/vault"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

func main() {
	// We use a custom flag set because Tekton's API has the default flag set
	// configured too, which makes our command help make no sense.
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	kubeconfig := fs.String("kubeconfig", "", "Path to kubeconfig file. Only required if running outside of a cluster")
	vaultAddr := fs.String("vault-addr", "http://localhost:8200", "address to the vault server")
	vaultToken := fs.String("vault-token", "", "token used to authenticate with the vault server")
	vaultEngineMount := fs.String("vault-engine-mount", "nebula", "the engine mount to craft paths from")
	gcsBucket := fs.String("gcs-bucket", "", "the GCS bucket to upload logs into")
	metadataServiceImage := fs.String("metadata-service-image", "gcr.io/nebula-235818/nebula-metadata-api:latest", "the image and tag to use for the metadata service api")
	metadataServiceImagePullSecret := fs.String("metadata-service-image-pull-secret", "", "the optionally namespaced name of the image pull secret to use for the metadata service")
	metadataServiceVaultAddr := fs.String("metadata-service-vault-addr", "", "the address to use when authenticating the metadata service to Vault")
	numWorkers := fs.Int("num-workers", 2, "the number of worker threads to spawn that process SecretAuth resources")

	fs.Parse(os.Args[1:])

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	cfg := &config.SecretAuthControllerConfig{
		Kubeconfig: *kubeconfig,

		MetadataServiceImage:           *metadataServiceImage,
		MetadataServiceImagePullSecret: *metadataServiceImagePullSecret,
		MetadataServiceVaultAddr:       *metadataServiceVaultAddr,
	}

	vc, err := vault.NewVaultAuth(*vaultAddr, *vaultToken, *vaultEngineMount)
	if err != nil {
		log.Fatal(err)
	}
	blobStorage, err := gcs.New(gcs.Options {
		BucketName: *gcsBucket,
	})
	if err != nil {
		log.Fatal(err)
	}

	controller, err := secretauth.NewController(cfg, vc, blobStorage)
	if err != nil {
		log.Fatal(err)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	go wait.Forever(klog.Flush, time.Second*2)
	go controller.Run(*numWorkers, stopCh)

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM)
	signal.Notify(termCh, syscall.SIGINT)
	<-termCh
}
