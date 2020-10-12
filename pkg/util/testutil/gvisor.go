package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallGVisorRuntimeClass(ctx context.Context, cl client.Client, handler string) error {
	return doInstall(ctx, cl, "gvisor", func(obj runtime.Object, gvk *schema.GroupVersionKind) {
		rc, ok := obj.(*nodev1beta1.RuntimeClass)
		if !ok || rc.GetName() != "runsc" {
			return
		}

		rc.Handler = handler
	})
}

func InstallGVisorRuntimeClass(t *testing.T, ctx context.Context, cl client.Client, handler string) {
	require.NoError(t, doInstallGVisorRuntimeClass(ctx, cl, handler))
}
