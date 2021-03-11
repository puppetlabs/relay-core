package testutil

import (
	"context"
	"testing"

	"github.com/puppetlabs/leg/k8sutil/pkg/manifest"
	"github.com/stretchr/testify/require"
	nodev1beta1 "k8s.io/api/node/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallGVisorRuntimeClass(t *testing.T, ctx context.Context, cl client.Client, handler string) {
	require.NoError(t, doInstall(ctx, cl, "gvisor", func(obj manifest.Object, gvk *schema.GroupVersionKind) {
		rc, ok := obj.(*nodev1beta1.RuntimeClass)
		if !ok || rc.GetName() != "runsc" {
			return
		}

		rc.Handler = handler
	}))
}
