package app_test

import (
	"context"
	"encoding/json"
	"path"
	"testing"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2/jwt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRunDepsConfigureAnnotate(t *testing.T) {
	ctx := context.Background()

	testutil.WithEndToEndEnvironment(t, ctx, []testutil.EndToEndEnvironmentInstaller{testutil.EndToEndEnvironmentWithPVPool}, func(e2e *testutil.EndToEndEnvironment) {
		e2e.WithTestNamespace(ctx, func(namespace *corev1.Namespace) {
			cl := e2e.ControllerClient

			tenant := &relayv1beta1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-tenant",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.TenantSpec{},
			}

			require.NoError(t, cl.Create(ctx, tenant))

			require.NoError(t, cl.Update(ctx, namespace))

			require.NoError(t, cl.Create(ctx, &relayv1beta1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-workflow",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.WorkflowSpec{
					TenantRef: corev1.LocalObjectReference{
						Name: "my-test-tenant",
					},
					Steps: []*relayv1beta1.Step{
						{
							Name: "my-test-step",
						},
					},
				},
			}))

			require.NoError(t, cl.Create(ctx, &relayv1beta1.Run{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-run",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.RunSpec{
					WorkflowRef: corev1.LocalObjectReference{
						Name: "my-test-workflow",
					},
				},
			}))

			run := obj.NewRun(client.ObjectKey{
				Namespace: namespace.Name,
				Name:      "my-test-run",
			})

			ok, err := run.Load(ctx, cl)
			require.NoError(t, err)
			require.True(t, ok)

			deps, err := app.ApplyRunDeps(ctx, cl, run, TestIssuer, TestMetadataAPIURL)
			require.NoError(t, err)

			ws := deps.Workflow.Object.Spec.Steps[0]

			var md metav1.ObjectMeta
			require.NoError(t, deps.AnnotateStepToken(ctx, &md, ws))

			tok := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
			require.NotEmpty(t, tok)

			var claims authenticate.Claims
			require.NoError(t, json.Unmarshal([]byte(tok), &claims))

			sat, err := deps.MetadataAPIServiceAccountTokenSecrets.DefaultTokenSecret.Token()
			require.NoError(t, err)
			require.NotEmpty(t, sat)

			assert.Equal(t, authenticate.ControllerIssuer, claims.Issuer)
			assert.Equal(t, jwt.Audience{authenticate.MetadataAPIAudienceV1}, claims.Audience)
			assert.Equal(t, path.Join("steps", app.ModelStep(run, ws).Hash().HexEncoding()), claims.Subject)
			assert.NotNil(t, claims.Expiry)
			assert.NotNil(t, claims.NotBefore)
			assert.NotNil(t, claims.IssuedAt)
			assert.Equal(t, namespace.Name, claims.KubernetesNamespaceName)
			assert.Equal(t, string(namespace.GetUID()), claims.KubernetesNamespaceUID)
			assert.Equal(t, sat, claims.KubernetesServiceAccountToken)
			assert.Equal(t, run.Object.GetName(), claims.RelayRunID)
			assert.Equal(t, ws.Name, claims.RelayName)
			assert.Equal(t, deps.ImmutableConfigMap.Key.Name, claims.RelayKubernetesImmutableConfigMapName)
			assert.Equal(t, deps.MutableConfigMap.Key.Name, claims.RelayKubernetesMutableConfigMapName)
		})
	})
}

// TODO: merge this test with the above using a case table.
func TestRunDepsConfigureWorkflowExecutionSink(t *testing.T) {
	ctx := context.Background()

	testutil.WithEndToEndEnvironment(t, ctx, []testutil.EndToEndEnvironmentInstaller{testutil.EndToEndEnvironmentWithPVPool}, func(e2e *testutil.EndToEndEnvironment) {
		e2e.WithTestNamespace(ctx, func(namespace *corev1.Namespace) {
			cl := e2e.ControllerClient

			token := "test-token"

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace.Name,
					Name:      "my-test-tenant",
				},
				StringData: map[string]string{
					"token": token,
				},
			}

			require.NoError(t, cl.Create(ctx, secret))

			tenant := &relayv1beta1.Tenant{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-tenant",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.TenantSpec{
					WorkflowExecutionSink: relayv1beta1.WorkflowExecutionSink{
						API: &relayv1beta1.APIWorkflowExecutionSink{
							URL: "https://unit-testing.relay.sh/workflow-run",
							TokenFrom: &relayv1beta1.APITokenSource{
								SecretKeyRef: &relayv1beta1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: secret.GetName(),
									},
									Key: "token",
								},
							},
						},
					},
				},
			}

			require.NoError(t, cl.Create(ctx, tenant))

			require.NoError(t, cl.Update(ctx, namespace))

			require.NoError(t, cl.Create(ctx, &relayv1beta1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-workflow",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.WorkflowSpec{
					TenantRef: corev1.LocalObjectReference{
						Name: "my-test-tenant",
					},
					Steps: []*relayv1beta1.Step{
						{
							Name: "my-test-step",
						},
					},
				},
			}))

			require.NoError(t, cl.Create(ctx, &relayv1beta1.Run{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-test-run",
					Namespace: namespace.Name,
				},
				Spec: relayv1beta1.RunSpec{
					WorkflowRef: corev1.LocalObjectReference{
						Name: "my-test-workflow",
					},
				},
			}))

			run := obj.NewRun(client.ObjectKey{
				Namespace: namespace.Name,
				Name:      "my-test-run",
			})

			ok, err := run.Load(ctx, cl)
			require.NoError(t, err)
			require.True(t, ok)

			deps, err := app.ApplyRunDeps(ctx, cl, run, TestIssuer, TestMetadataAPIURL)
			require.NoError(t, err)

			ws := deps.Workflow.Object.Spec.Steps[0]

			var md metav1.ObjectMeta
			require.NoError(t, deps.AnnotateStepToken(ctx, &md, ws))

			tok := md.GetAnnotations()[authenticate.KubernetesTokenAnnotation]
			require.NotEmpty(t, tok)

			var claims authenticate.Claims
			require.NoError(t, json.Unmarshal([]byte(tok), &claims))

			assert.Equal(t, token, claims.RelayWorkflowExecutionAPIToken)
			assert.Equal(t, tenant.Spec.WorkflowExecutionSink.API.URL, claims.RelayWorkflowExecutionAPIURL.URL.String())
		})
	})
}
