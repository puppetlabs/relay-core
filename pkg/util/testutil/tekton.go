package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallTektonPipeline(ctx context.Context, cl client.Client) error {
	return doInstallAndWait(ctx, cl, "tekton-pipelines", "tekton")
}

func InstallTektonPipeline(t *testing.T, ctx context.Context, cl client.Client) {
	require.NoError(t, doInstallTektonPipeline(ctx, cl))
}
