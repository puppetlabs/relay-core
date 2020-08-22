package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func doInstallAmbassador(ctx context.Context, cl client.Client, mapper meta.RESTMapper) error {
	// Ambassador requires Knative Serving; it won't detect it after the fact.
	if err := doInstallKnativeServing(ctx, cl); err != nil {
		return err
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ambassador-webhook",
		},
	}
	if err := cl.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	var patchers []ParseKubernetesManifestPatcherFunc
	patchers = append(patchers, KubernetesDefaultNamespacePatcher(mapper, ns.GetName()))
	patchers = append(patchers, func(obj runtime.Object, gvk *schema.GroupVersionKind) {
		deployment, ok := obj.(*appsv1.Deployment)
		if !ok || deployment.GetName() != "ambassador" {
			return
		}

		for i, c := range deployment.Spec.Template.Spec.Containers {
			if c.Name != "ambassador" {
				continue
			}

			SetKubernetesEnvVar(&c.Env, "AMBASSADOR_ID", "webhook")
			SetKubernetesEnvVar(&c.Env, "AMBASSADOR_KNATIVE_SUPPORT", "true")

			deployment.Spec.Template.Spec.Containers[i] = c
		}

		// Make as minimal as possible for testing.
		deployment.Spec.Replicas = func(i int32) *int32 { return &i }(1)
		deployment.Spec.RevisionHistoryLimit = func(i int32) *int32 { return &i }(0)

		// Don't allow old pods to linger.
		deployment.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
		deployment.Spec.Strategy.RollingUpdate = nil
	})

	if err := doInstallAndWait(ctx, cl, ns.GetName(), "ambassador", patchers...); err != nil {
		return err
	}

	return nil
}

func InstallAmbassador(t *testing.T, ctx context.Context, cl client.Client, mapper meta.RESTMapper) {
	require.NoError(t, doInstallAmbassador(ctx, cl, mapper))
}
