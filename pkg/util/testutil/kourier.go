package testutil

import (
	"context"
	"testing"

	"github.com/puppetlabs/leg/k8sutil/pkg/manifest"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallKourier(ctx context.Context, cl client.Client, mapper meta.RESTMapper) error {
	// Kourier requires Knative Serving; it won't detect it after the fact.
	if err := doInstallKnativeServing(ctx, cl); err != nil {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kourier-system",
		},
	}
	if err := cl.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	var patchers []manifest.PatcherFunc
	patchers = append(patchers, manifest.DefaultNamespacePatcher(mapper, ns.GetName()))

	if err := doInstallAndWait(ctx, cl, ns.GetName(), "kourier", patchers...); err != nil {
		return err
	}

	return nil
}

func InstallKourier(t *testing.T, ctx context.Context, cl client.Client, mapper meta.RESTMapper) {
	require.NoError(t, doInstallKourier(ctx, cl, mapper))
}
