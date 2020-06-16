package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/puppetlabs/nebula-libs/storage/file/v2"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/retry"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func assertWebhookTriggerResponseContains(t *testing.T, ctx context.Context, expected string, wt *relayv1beta1.WebhookTrigger) {
	// Wait for trigger to settle in Knative and pull its URL.
	var targetURL string
	require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
		if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
			Namespace: wt.GetNamespace(),
			Name:      wt.GetName(),
		}, wt); err != nil {
			return retry.RetryPermanent(err)
		}

		for _, cond := range wt.Status.Conditions {
			if cond.Type != relayv1beta1.WebhookTriggerReady {
				continue
			} else if cond.Status != corev1.ConditionTrue {
				break
			}

			targetURL = wt.Status.URL
			return retry.RetryPermanent(nil)
		}

		return retry.RetryTransient(fmt.Errorf("waiting for webhook trigger to be successfully created"))
	}))
	require.NotEmpty(t, targetURL)

	code, stdout, stderr := testutil.RunScriptInAlpine(
		t, ctx, e2e.RESTConfig, e2e.Interface,
		metav1.ObjectMeta{
			Namespace: wt.GetNamespace(),
		},
		fmt.Sprintf("exec wget -q -O - %s", targetURL),
	)
	assert.Equal(t, 0, code, "unexpected error from script: standard output:\n%s\n\nstandard error:\n%s", stdout, stderr)
	assert.Contains(t, stdout, expected)
}

func TestWebhookTriggerServesResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWebhookTriggerReconciler,
	}, func(cfg *Config) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", cfg.Namespace.GetName()),
					},
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))

		// Wait for TenantReady.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
				Namespace: tn.GetNamespace(),
				Name:      tn.GetName(),
			}, tn); err != nil {
				return retry.RetryPermanent(err)
			}

			for _, cond := range tn.Status.Conditions {
				if cond.Type != relayv1beta1.TenantReady {
					continue
				} else if cond.Status != corev1.ConditionTrue {
					break
				}

				return retry.RetryPermanent(nil)
			}

			return retry.RetryTransient(fmt.Errorf("waiting for tenant to be successfully created"))
		}))

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Image: "hashicorp/http-echo",
				Args: []string{
					"-listen", ":8080",
					"-text", "Hello, Relay!",
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, "Hello, Relay!", wt)
	})
}

func TestWebhookTriggerScript(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWebhookTriggerReconciler,
	}, func(cfg *Config) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", cfg.Namespace.GetName()),
					},
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))

		// Wait for TenantReady.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
				Namespace: tn.GetNamespace(),
				Name:      tn.GetName(),
			}, tn); err != nil {
				return retry.RetryPermanent(err)
			}

			for _, cond := range tn.Status.Conditions {
				if cond.Type != relayv1beta1.TenantReady {
					continue
				} else if cond.Status != corev1.ConditionTrue {
					break
				}

				return retry.RetryPermanent(nil)
			}

			return retry.RetryTransient(fmt.Errorf("waiting for tenant to be successfully created"))
		}))

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Image: "alpine:latest",
				Input: []string{
					"apk --no-cache add socat",
					`exec socat TCP-LISTEN:$PORT,crlf,reuseaddr,fork SYSTEM:'echo "HTTP/1.1 200 OK"; echo "Connection: close"; echo; echo "Hello, Relay!";'`,
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, "Hello, Relay!", wt)
	})
}

func TestWebhookTriggerHasAccessToMetadataAPI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPI,
		ConfigWithWebhookTriggerReconciler,
	}, func(cfg *Config) {
		// Set a secret and connection for this webhook trigger to look up.
		cfg.Vault.SetSecret(t, "my-tenant-id", "foo", "Hello")
		cfg.Vault.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
			"accessKeyID":     "AKIA123456789",
			"secretAccessKey": "very-nice-key",
		})

		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				TriggerEventSink: relayv1beta1.TriggerEventSink{
					API: &relayv1beta1.APITriggerEventSink{
						URL:   s.URL,
						Token: "foobarbaz",
					},
				},
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", cfg.Namespace.GetName()),
					},
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))

		// Wait for TenantReady.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
				Namespace: tn.GetNamespace(),
				Name:      tn.GetName(),
			}, tn); err != nil {
				return retry.RetryPermanent(err)
			}

			for _, cond := range tn.Status.Conditions {
				if cond.Type != relayv1beta1.TenantReady {
					continue
				} else if cond.Status != corev1.ConditionTrue {
					break
				}

				return retry.RetryPermanent(nil)
			}

			return retry.RetryTransient(fmt.Errorf("waiting for tenant to be successfully created"))
		}))

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            "my-tenant-id",
				},
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Image: "hashicorp/http-echo",
				Args: []string{
					"-listen", ":8080",
					"-text", "Hello, Relay!",
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
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

		// Issue a request to spin up a pod.
		assertWebhookTriggerResponseContains(t, ctx, "Hello, Relay!", wt)

		// Pull the pod and get its IP.
		pod := &corev1.Pod{}
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			pods := &corev1.PodList{}
			if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name), client.MatchingLabels{
				model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
			}); err != nil {
				return retry.RetryPermanent(err)
			}

			if len(pods.Items) == 0 {
				return retry.RetryTransient(fmt.Errorf("waiting for pod"))
			}

			pod = &pods.Items[0]
			if pod.Status.PodIP == "" {
				return retry.RetryTransient(fmt.Errorf("waiting for pod IP"))
			}

			return retry.RetryPermanent(nil)
		}))

		// Retrieve the spec.
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/spec", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result evaluate.JSONResultEnvelope
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, map[string]interface{}{
			"secret": "Hello",
			"connection": map[string]interface{}{
				"accessKeyID":     "AKIA123456789",
				"secretAccessKey": "very-nice-key",
			},
			"foo": "bar",
		}, result.Value.Data)

		// Dispatch an event.
		req.Method = http.MethodPost
		req.URL.Path = "/events"
		req.Body = ioutil.NopCloser(bytes.NewBufferString(`{"data":{"test":"value"}}`))

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusAccepted, resp.StatusCode)
		require.NotEqual(t, 0, reqs)
	})
}
