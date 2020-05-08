package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKnativeServing(ctx context.Context, cl client.Client, version string) error {
	return doInstall(ctx, cl, "knative", "knative-serving", version)
}

func InstallKnativeServing(t *testing.T, ctx context.Context, cl client.Client, version string) {
	require.NoError(t, doInstallKnativeServing(ctx, cl, version))
}
