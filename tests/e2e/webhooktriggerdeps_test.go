package e2e_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWebhookTriggerDepsConfigureAnnotate(t *testing.T) {
	ctx := context.Background()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, namespace *corev1.Namespace) {
		cl := eit.ControllerClient

		require.NoError(t, cl.Create(ctx, &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-tenant",
				Namespace: namespace.Name,
			},
			Spec: relayv1beta1.TenantSpec{},
		}))

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-trigger",
				Namespace: namespace.Name,
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				TenantRef: corev1.LocalObjectReference{
					Name: "my-test-tenant",
				},
				Name: "hello-world",
				Container: relayv1beta1.Container{
					Image: "alpine:latest",
				},
			},
		}
		require.NoError(t, cl.Create(ctx, wt))

		deps := app.NewWebhookTriggerDeps(obj.NewWebhookTriggerFromObject(wt), TestIssuer, TestMetadataAPIURL)
		err := retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			r, err := deps.Load(ctx, eit.ControllerClient)
			return r.All && err == nil, err
		})
		require.NoError(t, err)

		var md metav1.ObjectMeta
		require.NoError(t, deps.AnnotateTriggerToken(ctx, &md))

		tok1 := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
		require.NotEmpty(t, tok1)

		// Change something minor about the trigger; no reissue.
		Mutate(t, ctx, eit, wt, func() {
			wt.Spec.Image = "hashicorp/http-echo:latest"
		})

		deps = app.NewWebhookTriggerDeps(obj.NewWebhookTriggerFromObject(wt), TestIssuer, TestMetadataAPIURL)
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			r, err := deps.Load(ctx, eit.ControllerClient)
			return r.All && err == nil, err
		}))
		require.Equal(t, wt.Spec.Image, deps.WebhookTrigger.Object.Spec.Image)

		require.NoError(t, deps.AnnotateTriggerToken(ctx, &md))

		tok2 := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
		require.Equal(t, tok1, tok2)

		// Change the name of the trigger; reissue.
		Mutate(t, ctx, eit, wt, func() {
			wt.Spec.Name = "hello-whirled"
		})

		deps = app.NewWebhookTriggerDeps(obj.NewWebhookTriggerFromObject(wt), TestIssuer, TestMetadataAPIURL)
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			r, err := deps.Load(ctx, eit.ControllerClient)
			return r.All && err == nil, err
		}))
		require.Equal(t, wt.Spec.Name, deps.WebhookTrigger.Object.Spec.Name)

		require.NoError(t, deps.AnnotateTriggerToken(ctx, &md))

		tok3 := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
		require.NotEmpty(t, tok3)
		require.NotEqual(t, tok1, tok3)
	})
}
