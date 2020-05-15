package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallTektonPipeline(ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) error {
	return doInstall(ctx, cl, mapper, "tekton-pipelines", "tekton", version)
}

func InstallTektonPipeline(t *testing.T, ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) {
	require.NoError(t, doInstallTektonPipeline(ctx, cl, mapper, version))
}
