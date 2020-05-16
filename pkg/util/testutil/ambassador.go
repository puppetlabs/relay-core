package testutil

import (
	"context"
	"testing"

	goversion "github.com/hashicorp/go-version"
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

var MinimumAmbassadorVersion = goversion.Must(goversion.NewVersion("1.5.0"))

const AmbassadorTestImage = "gcr.io/nebula-contrib/ambassador:git-v1.4.2-75-g00e350c11"

func doInstallAmbassador(ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) error {
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

			// XXX: Remove this after upstream release v1.5.0! Gross!
			if goversion.Must(goversion.NewVersion(version)).LessThan(MinimumAmbassadorVersion) {
				// We have to swap out the image with our own image to get the
				// behavior we want.
				c.Image = AmbassadorTestImage
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

	if err := doInstall(ctx, cl, ns.GetName(), "ambassador", version, patchers...); err != nil {
		return err
	}

	return nil
}

func InstallAmbassador(t *testing.T, ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) {
	require.NoError(t, doInstallAmbassador(ctx, cl, mapper, version))
}
