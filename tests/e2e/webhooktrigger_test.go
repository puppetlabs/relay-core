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

	"github.com/google/uuid"
	"github.com/puppetlabs/leg/k8sutil/pkg/test/endtoend"
	_ "github.com/puppetlabs/leg/storage/file"
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog/v2"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForWebhookTriggerResponse(t *testing.T, ctx context.Context, cfg *Config, wt *relayv1beta1.WebhookTrigger) (int, string, string) {
	// Wait for trigger to settle in Knative and pull its URL.
	var targetURL string
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{
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
		cfg.Environment.Environment,
		fmt.Sprintf("exec wget -q -O - %s", targetURL),
		endtoend.ExecerWithNamespace(wt.GetNamespace()),
	)
	require.NoError(t, err)
	return r.Code, r.Stdout, r.Stderr
}

func assertWebhookTriggerResponseContains(t *testing.T, ctx context.Context, cfg *Config, expected string, wt *relayv1beta1.WebhookTrigger) {
	code, stdout, stderr := waitForWebhookTriggerResponse(t, ctx, cfg, wt)
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
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, cfg, "Hello, Relay!", wt)
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
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, cfg, "Hello, Relay!", wt)
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
		cfg.Vault.SetSecret(t, "my-tenant-id", "accessKeyId", "AKIA123456789")
		cfg.Vault.SetSecret(t, "my-tenant-id", "secretAccessKey", "that's-a-very-nice-key-you-have-there")
		cfg.Vault.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
			"accessKeyID":     "AKIA123456789",
			"secretAccessKey": "that's-a-very-nice-key-you-have-there",
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
		CreateAndWaitForTenant(t, ctx, cfg, tn)

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
				Container: relayv1beta1.Container{
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		// Issue a request to spin up a pod.
		assertWebhookTriggerResponseContains(t, ctx, cfg, "Hello, Relay!", wt)

		// Pull the pod and get its IP.
		pod := &corev1.Pod{}
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			pods := &corev1.PodList{}
			if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name), client.MatchingLabels{
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

		// Retrieve the spec.
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/spec", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var result exprmodel.JSONResultEnvelope
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, map[string]interface{}{
			"secret": "Hello",
			"connection": map[string]interface{}{
				"accessKeyID":     "AKIA123456789",
				"secretAccessKey": "that's-a-very-nice-key-you-have-there",
			},
			"foo": "bar",
		}, result.Value.Data)

		req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/environment", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, map[string]interface{}{
			"AWS_ACCESS_KEY_ID":     "AKIA123456789",
			"AWS_SECRET_ACCESS_KEY": "that's-a-very-nice-key-you-have-there",
		}, result.Value.Data)

		req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/environment/AWS_ACCESS_KEY_ID", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, "AKIA123456789", result.Value.Data)

		req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/environment/AWS_SECRET_ACCESS_KEY", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, "that's-a-very-nice-key-you-have-there", result.Value.Data)

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

func TestWebhookTriggerTenantUpdatePropagation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWebhookTriggerReconciler,
	}, func(cfg *Config) {
		child1 := fmt.Sprintf("%s-child-1", cfg.Namespace.GetName())
		child2 := fmt.Sprintf("%s-child-2", cfg.Namespace.GetName())

		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
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
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		// Create a webhook trigger. The Knative service will come up in the first
		// namespace.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		var ks servingv1.ServiceList
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.List(ctx, &ks, client.InNamespace(child1)); err != nil {
				return true, err
			}

			if len(ks.Items) == 0 {
				return false, fmt.Errorf("waiting for Knative service in first child namespace")
			}

			return true, nil
		}))

		// Change the tenant to use a new namespace. The Knative service should then
		// switch to the new namespace.
		Mutate(t, ctx, cfg, tn, func() {
			tn.Spec.NamespaceTemplate.Metadata.Name = child2
		})

		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.List(ctx, &ks, client.InNamespace(child2)); err != nil {
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

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWebhookTriggerReconciler,
	}, func(cfg *Config) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-child", cfg.Namespace.GetName()),
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		// Delete tenant first. This should pretty much break the webhook
		// reconciliation.
		require.NoError(t, cfg.Environment.ControllerClient.Delete(ctx, tn))
		require.NoError(t, testutil.WaitForObjectDeletion(ctx, cfg.Environment.ControllerClient, tn))

		// Webhook should still be deletable, though.
		require.NoError(t, cfg.Environment.ControllerClient.Delete(ctx, wt))
		require.NoError(t, testutil.WaitForObjectDeletion(ctx, cfg.Environment.ControllerClient, wt))
	})
}

func TestWebhookTriggerKnativeRevisions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		// This shouldn't settle because the given input is not sufficient to
		// satisfy Knative. We're just going to check to make sure the
		// respective revisions actually get created.
		revisions := &servingv1.RevisionList{}
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.List(ctx, revisions, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name)); err != nil {
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
		Mutate(t, ctx, cfg, wt, func() { wt.Spec.Input = []string{"echo goodbye"} })

		// We should shortly have two revisions.
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.List(ctx, revisions, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name)); err != nil {
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

func TestWebhookTriggerKnativeRevisionsWithTenantToolInjectionUsingInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithTenantReconciler,
		ConfigWithWebhookTriggerReconciler,
		ConfigWithVolumeClaimAdmission,
		ConfigWithMetadataAPIBoundInCluster,
	}, func(cfg *Config) {
		cfg.Vault.SetSecret(t, "my-tenant-id", "foo", "Relay")

		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant-" + uuid.New().String(),
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadOnlyMany,
							},
						},
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger-" + uuid.New().String(),
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
				Container: relayv1beta1.Container{
					Image: "alpine:latest",
					Input: []string{
						"apk --no-cache add socat",
						`exec socat TCP-LISTEN:$PORT,crlf,reuseaddr,fork SYSTEM:'echo "HTTP/1.1 200 OK"; echo "Connection: close"; echo; echo "Hello, $TEST_WHO!";'`,
					},
					Env: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
						"TEST_WHO": map[string]interface{}{
							"$type": "Secret",
							"name":  "foo",
						},
					}),
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, cfg, "Hello, Relay!", wt)

		pod := &corev1.Pod{}
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			pods := &corev1.PodList{}
			if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
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
	})
}

func TestWebhookTriggerKnativeRevisionsWithTenantToolInjectionUsingCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithTenantReconciler,
		ConfigWithWebhookTriggerReconciler,
		ConfigWithVolumeClaimAdmission,
	}, func(cfg *Config) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant-" + uuid.New().String(),
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadOnlyMany,
							},
						},
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger-" + uuid.New().String(),
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
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, cfg, "Hello, Relay!", wt)

		pod := &corev1.Pod{}
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			pods := &corev1.PodList{}
			if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
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
	})
}

func TestWebhookTriggerInGVisor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWebhookTriggerReconciler,
		ConfigWithPodEnforcementAdmission,
	}, func(cfg *Config) {
		if cfg.Environment.GVisorRuntimeClassName == "" {
			t.Skip("gVisor is not available on this platform")
		}

		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant",
				Namespace: cfg.Namespace.GetName(),
			},
		}
		CreateAndWaitForTenant(t, ctx, cfg, tn)

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Container: relayv1beta1.Container{
					Image: "alpine:latest",
					Input: []string{
						"apk --no-cache add socat",
						`exec socat TCP-LISTEN:$PORT,crlf,reuseaddr,fork SYSTEM:'echo "HTTP/1.1 200 OK"; echo "Connection: close"; echo; dmesg;'`,
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

		assertWebhookTriggerResponseContains(t, ctx, cfg, "gVisor", wt)
	})
}

func TestWebhookTriggerCheckoutGarbageCollection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	WithConfig(t, ctx, nil, func(cfg *Config) {
		tn := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-tenant-" + uuid.New().String(),
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{
					VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{
								corev1.ReadOnlyMany,
							},
						},
					},
				},
			},
		}

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger-" + uuid.New().String(),
				Namespace: cfg.Namespace.GetName(),
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

		var initialRevision *servingv1.Revision

		{
			end, _ := ctx.Deadline()
			ctx, cancel := context.WithTimeout(ctx, time.Until(end)/2)
			defer cancel()

			WithConfig(t, ctx, []ConfigOption{
				ConfigInNamespace(cfg.Namespace),
				ConfigWithToolInjectionPoolName("tools-1"),
				ConfigWithTenantReconciler,
				ConfigWithWebhookTriggerReconciler,
				ConfigWithVolumeClaimAdmission,
				ConfigWithoutCleanup,
			}, func(cfg *Config) {
				CreateAndWaitForTenant(t, ctx, cfg, tn)
				require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wt))

				// Wait for a revision and checkout to appear.
				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					revs := &servingv1.RevisionList{}
					if err := cfg.Environment.ControllerClient.List(ctx, revs, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
						model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
					}); err != nil {
						return true, err
					}

					if len(revs.Items) == 0 {
						return false, fmt.Errorf("waiting for revision")
					}

					initialRevision = &revs.Items[0]
					return true, nil
				}))

				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					cos := &pvpoolv1alpha1.CheckoutList{}
					if err := cfg.Environment.ControllerClient.List(ctx, cos, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
						model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
					}); err != nil {
						return true, err
					}

					if len(cos.Items) == 0 {
						return false, fmt.Errorf("waiting for checkout")
					}

					return true, nil
				}))
			})
		}

		{
			WithConfig(t, ctx, []ConfigOption{
				ConfigInNamespace(cfg.Namespace),
				ConfigWithToolInjectionPoolName("tools-2"),
				ConfigWithTenantReconciler,
				ConfigWithWebhookTriggerReconciler,
				ConfigWithVolumeClaimAdmission,
			}, func(cfg *Config) {
				// Wait for another revision and checkout because the pool name changed.
				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					revs := &servingv1.RevisionList{}
					if err := cfg.Environment.ControllerClient.List(ctx, revs, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
						model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
					}); err != nil {
						return true, err
					}

					switch len(revs.Items) {
					case 1:
						return false, fmt.Errorf("waiting for revision")
					case 2:
						return true, nil
					default:
						return true, fmt.Errorf("expected 1 or 2 revisions, got %d", len(revs.Items))
					}
				}))

				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					cos := &pvpoolv1alpha1.CheckoutList{}
					if err := cfg.Environment.ControllerClient.List(ctx, cos, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
						model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
					}); err != nil {
						return true, err
					}

					switch len(cos.Items) {
					case 1:
						return false, fmt.Errorf("waiting for checkout")
					case 2:
						return true, nil
					default:
						return true, fmt.Errorf("expected 1 or 2 checkouts, got %d", len(cos.Items))
					}
				}))

				// Delete the initial revision.
				require.NoError(t, cfg.Environment.ControllerClient.Delete(ctx, initialRevision))
				klog.InfoS("deleted initial revision", "revision", initialRevision.GetName())

				// We should now drop down to a single checkout.
				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					cos := &pvpoolv1alpha1.CheckoutList{}
					if err := cfg.Environment.ControllerClient.List(ctx, cos, client.InNamespace(tn.Status.Namespace), client.MatchingLabels{
						model.RelayControllerWebhookTriggerIDLabel: wt.GetName(),
					}); err != nil {
						return true, err
					}

					switch len(cos.Items) {
					case 1:
						return true, nil
					case 2:
						return false, fmt.Errorf("waiting for checkout to be deleted")
					default:
						return true, fmt.Errorf("expected 1 or 2 checkouts, got %d", len(cos.Items))
					}
				}))
			})
		}
	})
}
