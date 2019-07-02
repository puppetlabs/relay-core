package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controllers/secretauth"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to kubeconfig file. Only required if running outside of a cluster")

	flag.Parse()

	cfg := &config.SecretAuthControllerConfig{
		Kubeconfig: *kubeconfig,
	}

	controller, err := secretauth.NewController(cfg)
	if err != nil {
		log.Fatal(err)
	}

	stopCh := make(chan struct{})
	defer close(stopCh)

	go controller.Run(stopCh)

	termCh := make(chan os.Signal, 1)
	signal.Notify(termCh, syscall.SIGTERM)
	signal.Notify(termCh, syscall.SIGINT)
	<-termCh
}
