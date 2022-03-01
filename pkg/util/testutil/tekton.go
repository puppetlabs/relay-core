package testutil

import (
	"context"
	"testing"

	"github.com/puppetlabs/leg/k8sutil/pkg/manifest"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func InstallTektonPipeline(t *testing.T, ctx context.Context, cl client.Client) {
	var patchers []manifest.PatcherFunc
	patchers = append(patchers, func(obj manifest.Object, gvk *schema.GroupVersionKind) {
		crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
		if !ok {
			return
		}

		crd.Spec.PreserveUnknownFields = false
	})

	require.NoError(t, doInstallAndWait(ctx, cl, "tekton-pipelines", "tekton", patchers...))
}
