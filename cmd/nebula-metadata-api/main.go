package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/puppetlabs/horsehead/v2/mainutil"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/op"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server"
)

// defaultServiceAccountTokenPath is the default path to use for reading the service account
// JWT that is passed to vault for logging in.
const defaultServiceAccountTokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func main() {
	bindAddr := flag.String("bind-addr", "localhost:7000", "Host and port to bind the server to")
	debug := flag.Bool("debug", false, "Enable debug output")
	vaultAddr := flag.String("vault-addr", "http://localhost:8200", "Address to the vault server")
	vaultToken := flag.String("vault-token", "", "Specify in place of -vault-role and -service-account-token-path for using a basic vault token auth")
	vaultRole := flag.String("vault-role", "", "The role to use when logging into the vault server")
	serviceAccountTokenPath := flag.String("service-account-token-path",
		defaultServiceAccountTokenPath, "The path to k8s pod service account token")
	workflowID := flag.String("workflow-id", "", "The id of the workflow these secrets are scoped to")
	vaultEngineMount := flag.String("vault-engine-mount", "nebula", "The engine mount to use when crafting secret paths")
	namespace := flag.String("namespace", "", "The kubernetes namespace that contains the workflow")
	devPreConfigPath := flag.String("development-preconfiguration-path", "", "The path to a development preconfiguration file. This option will put the server in development mode and all managers will operate in in-memory mode.")

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
		DevelopmentPreConfigPath:   *devPreConfigPath,
		Logger:                     NewLogger(LoggerOptions{Debug: *debug}),
	}
	ctx := context.Background()

	var managers op.ManagerFactory
	var err errors.Error

	if cfg.DevelopmentPreConfigPath == "" {
		managers, err = op.NewForKubernetes(ctx, &cfg)
	} else {
		managers, err = op.NewForDev(ctx, &cfg)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	srv := server.New(&cfg, managers)

	os.Exit(mainutil.TrapAndWait(ctx, srv.Run))
}
