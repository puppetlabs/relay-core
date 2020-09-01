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

	_ "github.com/puppetlabs/horsehead/v2/storage/file"
	"github.com/puppetlabs/relay-core/pkg/admission"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/retry"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func waitForWebhookTriggerResponse(t *testing.T, ctx context.Context, wt *relayv1beta1.WebhookTrigger) (int, string, string) {
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

	return testutil.RunScriptInAlpine(
		t, ctx, e2e.RESTConfig, e2e.Interface,
		metav1.ObjectMeta{
			Namespace: wt.GetNamespace(),
		},
		fmt.Sprintf("exec wget -q -O - %s", targetURL),
	)
}

func assertWebhookTriggerResponseContains(t *testing.T, ctx context.Context, expected string, wt *relayv1beta1.WebhookTrigger) {
	code, stdout, stderr := waitForWebhookTriggerResponse(t, ctx, wt)
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
		CreateAndWaitForTenant(t, ctx, tn)

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
		CreateAndWaitForTenant(t, ctx, tn)

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
		CreateAndWaitForTenant(t, ctx, tn)

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
		CreateAndWaitForTenant(t, ctx, tn)

		// Create a webhook trigger. The Knative service will come up in the first
		// namespace.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Image: "alpine",
				Input: []string{
					"echo hi",
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

		var ks servingv1.ServiceList
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.List(ctx, &ks, client.InNamespace(child1)); err != nil {
				return retry.RetryPermanent(err)
			}

			if len(ks.Items) == 0 {
				return retry.RetryTransient(fmt.Errorf("waiting for Knative service in first child namespace"))
			}

			return retry.RetryPermanent(nil)
		}))

		// Change the tenant to use a new namespace. The Knative service should then
		// switch to the new namespace.
		Mutate(t, ctx, tn, func() {
			tn.Spec.NamespaceTemplate.Metadata.Name = child2
		})

		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.List(ctx, &ks, client.InNamespace(child2)); err != nil {
				return retry.RetryPermanent(err)
			}

			if len(ks.Items) == 0 {
				return retry.RetryTransient(fmt.Errorf("waiting for Knative service in second child namespace"))
			}

			return retry.RetryPermanent(nil)
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
		CreateAndWaitForTenant(t, ctx, tn)

		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Image: "alpine",
				Input: []string{
					"echo hi",
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

		// Delete tenant first. This should pretty much break the webhook
		// reconciliation.
		require.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, tn))
		require.NoError(t, testutil.WaitForObjectDeletion(ctx, e2e.ControllerRuntimeClient, tn))

		// Webhook should still be deletable, though.
		require.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, wt))
		require.NoError(t, testutil.WaitForObjectDeletion(ctx, e2e.ControllerRuntimeClient, wt))
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
		CreateAndWaitForTenant(t, ctx, tn)

		// Create a trigger.
		wt := &relayv1beta1.WebhookTrigger{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-trigger",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WebhookTriggerSpec{
				Image: "alpine",
				Input: []string{
					"echo hi",
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tn.GetName(),
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wt))

		// This shouldn't settle because the given input is not sufficient to
		// satisfy Knative. We're just going to check to make sure the
		// respective revisions actually get created.
		revisions := &servingv1.RevisionList{}
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.List(ctx, revisions, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name)); err != nil {
				return retry.RetryPermanent(err)
			}

			switch len(revisions.Items) {
			case 0:
				return retry.RetryTransient(fmt.Errorf("waiting for initial revision"))
			case 1:
				return retry.RetryPermanent(nil)
			default:
				return retry.RetryPermanent(fmt.Errorf("expected exactly 1 initial revision, got %d", len(revisions.Items)))
			}
		}))

		// Now we'll try to update the input to suggest to Knative to emit a new
		// revision.
		Mutate(t, ctx, wt, func() { wt.Spec.Input = []string{"echo goodbye"} })

		// We should shortly have two revisions.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.List(ctx, revisions, client.InNamespace(tn.Spec.NamespaceTemplate.Metadata.Name)); err != nil {
				return retry.RetryPermanent(err)
			}

			switch len(revisions.Items) {
			case 1:
				return retry.RetryTransient(fmt.Errorf("waiting for second revision"))
			case 2:
				return retry.RetryPermanent(nil)
			default:
				return retry.RetryPermanent(fmt.Errorf("expected exactly 2 final revisions, got %d", len(revisions.Items)))
			}
		}))
	})
}

func TestWebhookTriggerKnativeRevisionsWithTenantVolumeClaim(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithTenantReconciler,
		ConfigWithWebhookTriggerReconciler,
	}, func(cfg *Config) {
		hnd := testServerInjectorHandler{&webhook.Admission{Handler: admission.NewVolumeClaimHandler()}}
		cfg.Manager.SetFields(hnd)

		s := httptest.NewServer(hnd)
		defer s.Close()

		testutil.WithServiceBoundToHostHTTP(t, ctx, e2e.RESTConfig, e2e.Interface, s.URL, metav1.ObjectMeta{Namespace: cfg.Namespace.GetName()}, func(caPEM []byte, svc *corev1.Service) {
			handler := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
				TypeMeta: metav1.TypeMeta{
					APIVersion: admissionregistrationv1beta1.SchemeGroupVersion.Identifier(),
					Kind:       "MutatingWebhookConfiguration",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "volume-claim",
				},
				Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
					{
						Name: "volume-claim.admission.controller.relay.sh",
						ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
							Service: &admissionregistrationv1beta1.ServiceReference{
								Namespace: svc.GetNamespace(),
								Name:      svc.GetName(),
							},
							CABundle: caPEM,
						},
						Rules: []admissionregistrationv1beta1.RuleWithOperations{
							{
								Operations: []admissionregistrationv1beta1.OperationType{
									admissionregistrationv1beta1.Create,
									admissionregistrationv1beta1.Update,
								},
								Rule: admissionregistrationv1beta1.Rule{
									APIGroups:   []string{""},
									APIVersions: []string{"v1"},
									Resources:   []string{"pods"},
								},
							},
						},
						FailurePolicy: func(fp admissionregistrationv1beta1.FailurePolicyType) *admissionregistrationv1beta1.FailurePolicyType {
							return &fp
						}(admissionregistrationv1beta1.Fail),
						SideEffects: func(se admissionregistrationv1beta1.SideEffectClass) *admissionregistrationv1beta1.SideEffectClass {
							return &se
						}(admissionregistrationv1beta1.SideEffectClassNone),
						ReinvocationPolicy: func(rp admissionregistrationv1beta1.ReinvocationPolicyType) *admissionregistrationv1beta1.ReinvocationPolicyType {
							return &rp
						}(admissionregistrationv1beta1.IfNeededReinvocationPolicy),
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"testing.relay.sh/tools-volume-claim": "true",
							},
						},
					},
				},
			}

			require.NoError(t, e2e.ControllerRuntimeClient.Patch(ctx, handler, client.Apply, client.ForceOwnership, client.FieldOwner("relay-e2e")))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, handler))
			}()

			cfg.Vault.SetSecret(t, "my-tenant-id", "foo", "Hello")
			cfg.Vault.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
				"accessKeyID":     "AKIA123456789",
				"secretAccessKey": "very-nice-key",
			})

			size, _ := resource.ParseQuantity("50Mi")
			storageClassName := "relay-hostpath"
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
							Labels: map[string]string{
								"testing.relay.sh/tools-volume-claim": "true",
							},
						},
					},
					ToolInjection: relayv1beta1.ToolInjection{
						VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
							Spec: corev1.PersistentVolumeClaimSpec{
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceStorage: size,
									},
								},
								StorageClassName: &storageClassName,
							},
						},
					},
				},
			}
			CreateAndWaitForTenant(t, ctx, tn)

			var pvc corev1.PersistentVolumeClaim
			require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: tn.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tn.Status.Namespace}, &pvc))

			var pv corev1.PersistentVolume
			require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: tn.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany}, &pv))

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

			assertWebhookTriggerResponseContains(t, ctx, "Hello, Relay!", wt)

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

			e2e.ControllerRuntimeClient.Delete(ctx, &pvc)
			e2e.ControllerRuntimeClient.Delete(ctx, &pv)
		})
	})
}
