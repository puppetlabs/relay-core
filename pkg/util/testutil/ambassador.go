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

	if err := doInstall(ctx, cl, mapper, ns.GetName(), "ambassador", version); err != nil {
		return err
	}

	// Pull the deployment and patch it with the env vars we need.
	orig := &appsv1.Deployment{}
	if err := cl.Get(ctx, client.ObjectKey{Namespace: ns.GetName(), Name: "ambassador"}, orig); err != nil {
		return err
	}

	copy := orig.DeepCopy()
	for i, c := range copy.Spec.Template.Spec.Containers {
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

		copy.Spec.Template.Spec.Containers[i] = c
	}

	if err := cl.Patch(ctx, copy, client.MergeFrom(orig)); err != nil {
		return err
	}

	return nil
}

func InstallAmbassador(t *testing.T, ctx context.Context, cl client.Client, mapper meta.RESTMapper, version string) {
	require.NoError(t, doInstallAmbassador(ctx, cl, mapper, version))
}
