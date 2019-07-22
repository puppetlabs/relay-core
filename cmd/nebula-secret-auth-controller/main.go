package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controllers/secretauth"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets/vault"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file. Only required if running outside of a cluster")
	vaultAddr := flag.String("vault-addr", "http://localhost:8200", "address to the vault server")
	vaultToken := flag.String("vault-token", "", "token used to authenticate with the vault server")
	vaultEngineMount := flag.String("vault-engine-mount", "nebula", "the engine mount to craft paths from")
	metadataServiceImage := flag.String("metadata-service-image", "gcr.io/nebula-235818/nebula-metadata-api:latest", "the image and tag to use for the metadata service api")
	numWorkers := flag.Int("num-workers", 2, "the number of worker threads to spawn that process SecretAuth resources")

	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)

	cfg := &config.SecretAuthControllerConfig{
		Kubeconfig:           *kubeconfig,
		MetadataServiceImage: *metadataServiceImage,
	}

	vc, err := vault.NewVaultAuth(*vaultAddr, *vaultToken, *vaultEngineMount)
	if err != nil {
		log.Fatal(err)
	}

	controller, err := secretauth.NewController(cfg, vc)
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
