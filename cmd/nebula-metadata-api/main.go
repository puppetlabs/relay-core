package main

import (
	"context"
	"flag"
	"os"

	"github.com/puppetlabs/horsehead/mainutil"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server"
)

// defaultServiceAccountTokenPath is the default path to use for reading the service account
// JWT that is passed to vault for logging in.
const defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func main() {
	bindAddr := flag.String("bind-addr", "localhost:7000", "host and port to bind the server to")
	vaultAddr := flag.String("vault-addr", "http://localhost:8200", "address to the vault server")
	vaultToken := flag.String("vault-token", "", "Specify in place of -vault-role and -service-account-token-path for using a basic vault token auth")
	vaultRole := flag.String("vault-role", "", "the role to use when logging into the vault server")
	serviceAccountTokenPath := flag.String("service-account-token-path",
		defaultServiceAccountTokenPath, "the path to k8s pod service account token")
	workflowID := flag.String("workflow-id", "", "the id of the workflow these secrets are scoped to")
	vaultEngineMount := flag.String("vault-engine-mount", "nebula", "the engine mount to use when crafting secret paths")
	namespace := flag.String("namespace", "", "the kubernetes namespace that contains the workflow")

	flag.Parse()

	cfg := config.MetadataServerConfig{
		BindAddr:                   *bindAddr,
		VaultAddr:                  *vaultAddr,
		VaultRole:                  *vaultRole,
		VaultEngineMount:           *vaultEngineMount,
		VaultToken:                 *vaultToken,
		K8sServiceAccountTokenPath: *serviceAccountTokenPath,
		WorkflowID:                 *workflowID,
		Namespace:                  *namespace,
	}

	managers := op.NewDefaultManagerFactory(&cfg)

	srv := server.New(&cfg, managers)

	os.Exit(mainutil.TrapAndWait(context.Background(), srv.Run))
}
