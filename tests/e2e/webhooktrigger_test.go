package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/puppetlabs/leg/k8sutil/pkg/app/exec"
	"github.com/puppetlabs/leg/k8sutil/pkg/app/tunnel"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/test/endtoend"
	_ "github.com/puppetlabs/leg/storage/file"
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForWebhookTriggerResponse(t *testing.T, ctx context.Context, eit *EnvironmentInTest, wt *relayv1beta1.WebhookTrigger) (int, string, string) {
	// Wait for trigger to settle in Knative and pull its URL.
	var targetURL string
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := eit.ControllerClient.Get(ctx, client.ObjectKey{
			Namespace: wt.GetNamespace(),
			Name:      wt.GetName(),
		}, wt); err != nil {
			return true, err
		}

		for _, cond := range wt.Status.Conditions {
			if cond.Type != relayv1beta1.WebhookTriggerReady {
				continue
			} else if cond.Status != corev1.ConditionTrue {
				break
			}

			targetURL = wt.Status.URL
			return true, nil
		}

		return false, fmt.Errorf("waiting for webhook trigger to be successfully created")
	}, retry.WithBackoffFactory(backoff.Build(backoff.Linear(50*time.Millisecond)))))
	require.NotEmpty(t, targetURL)

	r, err := endtoend.Exec(
		ctx,
		eit.Environment,
		fmt.Sprintf("exec wget -q -O - %s", targetURL),
		endtoend.ExecerWithNamespace(wt.GetNamespace()),
	)
	require.NoError(t, err)
	return r.ExitCode, r.Stdout, r.Stderr
}

func assertWebhookTriggerResponseContains(t *testing.T, ctx context.Context, eit *EnvironmentInTest, expected string, wt *relayv1beta1.WebhookTrigger) {
	code, stdout, stderr := waitForWebhookTriggerResponse(t, ctx, eit, wt)
	assert.Equal(t, 0, code, "unexpected error from script: standard output:\n%s\n\nstandard error:\n%s", stdout, stderr)
	assert.Contains(t, stdout, expected)
}

func TestWebhookTriggerServesResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", ns.GetName()),
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tn.GetName(),
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "hashicorp/http-echo",
					Args: []string{
						"-listen", ":8080",
						"-text", "Hello, Relay!",
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, eit, "Hello, Relay!", wt)
	})
}

func TestWebhookTriggerScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", ns.GetName()),
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tn.GetName(),
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "alpine:latest",
					Input: []string{
						"apk --no-cache add socat",
						`exec socat TCP-LISTEN:$PORT,crlf,reuseaddr,fork SYSTEM:'echo "HTTP/1.1 200 OK"; echo "Connection: close"; echo; echo "Hello, Relay!";'`,
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, eit, "Hello, Relay!", wt)
	})
}

func TestWebhookTriggerHasAccessToMetadataAPI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var reqs int
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs++

		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer foobarbaz", r.Header.Get("Authorization"))

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]interface{}{
			"source": map[string]interface{}{
				"type":    "trigger",
				"trigger": map[string]interface{}{"name": "my-trigger"},
			},
			"data": map[string]interface{}{"test": "value"},
		}, body)

		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		st, err := tunnel.ApplyHTTP(ctx, eit.ControllerClient, client.ObjectKey{
			Namespace: ns.GetName(),
			Name:      "events",
		})
		require.NoError(t, err)

		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tenant-" + uuid.NewString(),
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				TriggerEventSink: relayv1beta1.TriggerEventSink{
					API: &relayv1beta1.APITriggerEventSink{
						URL:   st.URL(),
						Token: "foobarbaz",
					},
				},
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", ns.GetName()),
					},
				},
			},
		}

		// Set a secret and connection for this webhook trigger to look up.
		WithVault(t, ctx, eit, func(v *Vault) {
			v.SetSecret(t, ctx, tn.GetName(), "foo", "Hello")
			v.SetSecret(t, ctx, tn.GetName(), "accessKeyId", "AKIA123456789")
			v.SetSecret(t, ctx, tn.GetName(), "secretAccessKey", "that's-a-very-nice-key-you-have-there")
			v.SetConnection(t, ctx, ns.GetName(), "aws", "test", map[string]string{
				"accessKeyID":     "AKIA123456789",
				"secretAccessKey": "that's-a-very-nice-key-you-have-there",
			})
		})

		CreateAndWaitForTenant(t, ctx, eit, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    TestVaultEngineTenantPath,
					model.RelayVaultConnectionPathAnnotation: "connections/" + ns.GetName(),
					model.RelayVaultSecretPathAnnotation:     "workflows/" + tn.GetName(),
					model.RelayDomainIDAnnotation:            ns.GetName(),
					model.RelayTenantIDAnnotation:            tn.GetName(),
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "alpine:latest",
					Input: []string{
						"apk --no-cache add socat",
						`exec socat TCP-LISTEN:$PORT,crlf,reuseaddr,fork SYSTEM:'echo "HTTP/1.1 200 OK"; echo "Connection: close"; echo; echo "Hello, Relay!";'`,
					},
					Spec: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
						"secret": map[string]interface{}{
							"$type": "Secret",
							"name":  "foo",
						},
						"connection": map[string]interface{}{
							"$type": "Connection",
							"type":  "aws",
							"name":  "test",
						},
						"foo": "bar",
					}),
					Env: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
						"AWS_ACCESS_KEY_ID": map[string]interface{}{
							"$type": "Secret",
							"name":  "accessKeyId",
						},
						"AWS_SECRET_ACCESS_KEY": map[string]interface{}{
							"$type": "Secret",
							"name":  "secretAccessKey",
						},
					}),
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, wt))

		// Issue a request to spin up a pod.
		assertWebhookTriggerResponseContains(t, ctx, eit, "Hello, Relay!", wt)

		// Pull the pod and get its IP.
		pod := &corev1.Pod{}
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			pods := &corev1.PodList{}
			if err := eit.ControllerClient.List(ctx, pods, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name), client.MatchingLabels{
				model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
			}); err != nil {
				return true, err
			}

			if len(pods.Items) == 0 {
				return false, fmt.Errorf("waiting for pod")
			}

			pod = &pods.Items[0]
			if pod.Status.PodIP == "" {
				return false, fmt.Errorf("waiting for pod IP")
			}

			return true, nil
		}))

		var result api.GetSpecResponseEnvelope
		evaluateRequest := func(url string) api.GetSpecResponseEnvelope {
			r, err := exec.ShellScript(ctx, eit.RESTConfig, corev1obj.NewPodFromObject(pod), fmt.Sprintf("exec wget -q -O - %s", url), exec.WithContainer(wt.GetName()))
			require.NoError(t, err)
			require.Equal(t, 0, r.ExitCode, "unexpected error from script: standard output:\n%s\n\nstandard error:\n%s", r.Stdout, r.Stderr)

			require.NoError(t, json.Unmarshal([]byte(r.Stdout), &result))
			assert.True(t, result.Complete)

			return result
		}

		// Retrieve the spec.
		result = evaluateRequest(fmt.Sprintf("$%s/spec", model.EnvironmentVariableMetadataAPIURL))
		assert.Equal(t, map[string]interface{}{
			"secret": "Hello",
			"connection": map[string]interface{}{
				"accessKeyID":     "AKIA123456789",
				"secretAccessKey": "that's-a-very-nice-key-you-have-there",
			},
			"foo": "bar",
		}, result.Value.Data)

		result = evaluateRequest(fmt.Sprintf("$%s/environment", model.EnvironmentVariableMetadataAPIURL))
		assert.Equal(t, map[string]interface{}{
			"AWS_ACCESS_KEY_ID":     "AKIA123456789",
			"AWS_SECRET_ACCESS_KEY": "that's-a-very-nice-key-you-have-there",
		}, result.Value.Data)

		result = evaluateRequest(fmt.Sprintf("$%s/environment/AWS_ACCESS_KEY_ID", model.EnvironmentVariableMetadataAPIURL))
		assert.Equal(t, "AKIA123456789", result.Value.Data)

		result = evaluateRequest(fmt.Sprintf("$%s/environment/AWS_SECRET_ACCESS_KEY", model.EnvironmentVariableMetadataAPIURL))
		assert.Equal(t, "that's-a-very-nice-key-you-have-there", result.Value.Data)

		// Dispatch an event.
		err = tunnel.WithHTTPConnection(ctx, eit.RESTConfig, st, s.URL, func(ctx context.Context) {
			script := fmt.Sprintf(`exec wget -q --post-data '{"data":{"test":"value"}}' --header 'Content-Type: application/json' -O - $%s/events`, model.EnvironmentVariableMetadataAPIURL)

			r, err := exec.ShellScript(ctx, eit.RESTConfig, corev1obj.NewPodFromObject(pod), script, exec.WithContainer(wt.GetName()))
			require.NoError(t, err)
			require.Equal(t, 0, r.ExitCode, "unexpected error from script: standard output:\n%s\n\nstandard error:\n%s", r.Stdout, r.Stderr)
		})
		require.NoError(t, err)
		require.NotEqual(t, 0, reqs)
	})
}

func TestWebhookTriggerTenantUpdatePropagation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		child1 := fmt.Sprintf("%s-child-1", ns.GetName())
		child2 := fmt.Sprintf("%s-child-2", ns.GetName())

		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: child1,
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tn)

		// Create a webhook trigger. The Knative service will come up in the first
		// namespace.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tn.GetName(),
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "alpine",
					Input: []string{
						"echo hi",
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, wt))

		var ks servingv1.ServiceList
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.List(ctx, &ks, client.InNamespace(child1)); err != nil {
				return true, err
			}

			if len(ks.Items) == 0 {
				return false, fmt.Errorf("waiting for Knative service in first child namespace")
			}

			return true, nil
		}))

		// Change the tenant to use a new namespace. The Knative service should then
		// switch to the new namespace.
		Mutate(t, ctx, eit, tn, func() {
			tn.Spec.NamespaceTemplate = relayv1beta1.NamespaceTemplate{
				Metadata: metav1.ObjectMeta{
					Name: child2,
				},
			}
		})

		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.List(ctx, &ks, client.InNamespace(child2)); err != nil {
				return true, err
			}

			if len(ks.Items) == 0 {
				return false, fmt.Errorf("waiting for Knative service in second child namespace")
			}

			return true, nil
		}))
	})
}

func TestWebhookTriggerDeletionAfterTenantDeletion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", ns.GetName()),
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tn)

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tn.GetName(),
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "alpine",
					Input: []string{
						"echo hi",
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, wt))

		// Delete tenant first. This should pretty much break the webhook
		// reconciliation.
		require.NoError(t, eit.ControllerClient.Delete(ctx, tn))
		require.NoError(t, WaitForObjectDeletion(ctx, eit, tn))

		// Webhook should still be deletable, though.
		require.NoError(t, eit.ControllerClient.Delete(ctx, wt))
		require.NoError(t, WaitForObjectDeletion(ctx, eit, wt))
	})
}

func TestWebhookTriggerKnativeRevisions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", ns.GetName()),
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tn.GetName(),
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "alpine",
					Input: []string{
						"echo hi",
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, wt))

		// This shouldn't settle because the given input is not sufficient to
		// satisfy Knative. We're just going to check to make sure the
		// respective revisions actually get created.
		revisions := &servingv1.RevisionList{}
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.List(ctx, revisions, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name)); err != nil {
				return true, err
			}

			switch len(revisions.Items) {
			case 0:
				return false, fmt.Errorf("waiting for initial revision")
			case 1:
				return true, nil
			default:
				return true, fmt.Errorf("expected exactly 1 initial revision, got %d", len(revisions.Items))
			}
		}))

		// Now we'll try to update the input to suggest to Knative to emit a new
		// revision.
		Mutate(t, ctx, eit, wt, func() { wt.Spec.Input = []string{"echo goodbye"} })

		// We should shortly have two revisions.
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.List(ctx, revisions, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name)); err != nil {
				return true, err
			}

			switch len(revisions.Items) {
			case 1:
				return false, fmt.Errorf("waiting for second revision")
			case 2:
				return true, nil
			default:
				return true, fmt.Errorf("expected exactly 2 final revisions, got %d", len(revisions.Items))
			}
		}))
	})
}
