package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controllers/secretauth"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets/vault"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file. Only required if running outside of a cluster")
	vaultAddr := flag.String("vault-addr", "http://localhost:8200", "address to the vault server")
	vaultToken := flag.String("vault-token", "", "token used to authenticate with the vault server")
	vaultEngineMount := flag.String("vault-engine-mount", "nebula", "the engine mount to craft paths from")

	flag.Parse()

	cfg := &config.SecretAuthControllerConfig{
		Kubeconfig: *kubeconfig,
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

	go controller.Run(2, stopCh)

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM)
	signal.Notify(termCh, syscall.SIGINT)
	<-termCh
}
