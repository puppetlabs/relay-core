package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallPVPool(t *testing.T, ctx context.Context, cl client.Client) {
	require.NoError(t, doInstallAndWait(ctx, cl, "pvpool", "pvpool"))
}
