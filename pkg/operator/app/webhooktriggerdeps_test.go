package app_test

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWebhookTriggerDepsConfigureAnnotate(t *testing.T) {
	ctx := context.Background()

	testutil.WithEndToEndEnvironment(t, ctx, []testutil.EndToEndEnvironmentInstaller{testutil.EndToEndEnvironmentWithPVPool}, func(e2e *testutil.EndToEndEnvironment) {
		e2e.WithTestNamespace(ctx, func(namespace *corev1.Namespace) {
			cl := e2e.ControllerClient

			require.NoError(t, cl.Create(ctx, &relayv1beta1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-tenant",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.TenantSpec{},
			}))

			require.NoError(t, cl.Create(ctx, &relayv1beta1.WebhookTrigger{
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
			}))

			trigger := obj.NewWebhookTrigger(client.ObjectKey{
				Namespace: namespace.Name,
				Name:      "my-test-trigger",
			})

			ok, err := trigger.Load(ctx, cl)
			require.NoError(t, err)
			require.True(t, ok)

			deps, err := ApplyWebhookTriggerDeps(ctx, cl, trigger, TestIssuer, TestMetadataAPIURL)
			require.NoError(t, err)

			var md metav1.ObjectMeta
			require.NoError(t, deps.AnnotateTriggerToken(ctx, &md))

			tok1 := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
			require.NotEmpty(t, tok1)

			// Change something minor about the trigger; no reissue.
			trigger.Object.Spec.Image = "hashicorp/http-echo:latest"
			require.NoError(t, trigger.Persist(ctx, cl))

			deps, err = ApplyWebhookTriggerDeps(ctx, cl, trigger, TestIssuer, TestMetadataAPIURL)
			require.NoError(t, err)
			require.Equal(t, trigger.Object.Spec.Image, deps.WebhookTrigger.Object.Spec.Image)

			require.NoError(t, deps.AnnotateTriggerToken(ctx, &md))

			tok2 := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
			require.Equal(t, tok1, tok2)

			// Change the name of the trigger; reissue.
			trigger.Object.Spec.Name = "hello-whirled"
			require.NoError(t, trigger.Persist(ctx, cl))

			deps, err = ApplyWebhookTriggerDeps(ctx, cl, trigger, TestIssuer, TestMetadataAPIURL)
			require.NoError(t, err)
			require.Equal(t, trigger.Object.Spec.Image, deps.WebhookTrigger.Object.Spec.Image)

			require.NoError(t, deps.AnnotateTriggerToken(ctx, &md))

			tok3 := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
			require.NotEmpty(t, tok3)
			require.NotEqual(t, tok1, tok3)
		})
	})
}

func ApplyWebhookTriggerDeps(ctx context.Context, cl client.Client, wt *obj.WebhookTrigger, issuer authenticate.Issuer, metadataAPIURL *url.URL) (*app.WebhookTriggerDeps, error) {
	deps := app.NewWebhookTriggerDeps(wt, issuer, metadataAPIURL)

	if loaded, err := deps.Load(ctx, cl); err != nil {
		return nil, err
	} else if !loaded.Upstream {
		return nil, fmt.Errorf("waiting on WebhookTrigger upstream dependencies")
	}

	if err := app.ConfigureWebhookTriggerDeps(ctx, deps); err != nil {
		return nil, err
	}

	if err := deps.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return deps, nil
}
