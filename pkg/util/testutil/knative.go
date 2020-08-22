package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKnativeServing(ctx context.Context, cl client.Client) error {
	return doInstallAndWait(ctx, cl, "knative-serving", "knative")
}

func InstallKnativeServing(t *testing.T, ctx context.Context, cl client.Client) {
	require.NoError(t, doInstallKnativeServing(ctx, cl))
}
