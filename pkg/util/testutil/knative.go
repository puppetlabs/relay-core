package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKnativeServing(ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) error {
	return doInstall(ctx, cl, mapper, "knative-serving", "knative", version)
}

func InstallKnativeServing(t *testing.T, ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) {
	require.NoError(t, doInstallKnativeServing(ctx, cl, mapper, version))
}
