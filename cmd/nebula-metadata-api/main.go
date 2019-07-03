package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/puppetlabs/horsehead/mainutil"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/data/secrets/vault"
	"github.com/puppetlabs/nebula-tasks/pkg/server"
)

const defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func main() {
	bindAddr := flag.String("bind-addr", "localhost:7000", "host and port to bind the server to")
	vaultAddr := flag.String("vault-addr", "http://localhost:8200", "address to the vault server")
	vaultRole := flag.String("vault-role", "", "the role to use when logging into the vault server")
	serviceAccountTokenPath := flag.String("service-account-token-path",
		defaultServiceAccountTokenPath, "the path to k8s pod service account token")
	workflowName := flag.String("workflow-name", "", "the name of the workflow these secrets are scoped to")
	vaultEngineMount := flag.String("vault-engine-mount", "nebula", "the engine mount to use when crafting secret paths")

	flag.Parse()

	cfg := config.MetadataServerConfig{
		BindAddr: *bindAddr,
	}

	vc, err := vault.NewVaultWithKubernetesAuth(&vault.Config{
		Addr:                       *vaultAddr,
		K8sServiceAccountTokenPath: *serviceAccountTokenPath,
		Role:                       *vaultRole,
		WorkflowName:               *workflowName,
		EngineMount:                *vaultEngineMount,
	})
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(&cfg, vc)

	os.Exit(mainutil.TrapAndWait(context.Background(), srv.Run))
}
