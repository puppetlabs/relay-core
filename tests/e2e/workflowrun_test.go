package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/expr/evaluate"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/puppetlabs/nebula-tasks/pkg/obj"
	"github.com/puppetlabs/nebula-tasks/pkg/util/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestWorkflowRun tests that an instance of the controller, when given a run to
// process, correctly sets up a Tekton pipeline and that the resulting pipeline
// should be able to access a metadata API service.
func TestWorkflowRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPI,
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
		// Set a secret and connection for this workflow to look up.
		cfg.Vault.SetSecret(t, "my-tenant-id", "foo", "Hello")
		cfg.Vault.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
			"accessKeyID":     "AKIA123456789",
			"secretAccessKey": "very-nice-key",
		})

		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "my-test-run",
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            "my-tenant-id",
				},
			},
			Spec: nebulav1.WorkflowRunSpec{
				Name: "my-workflow-run-1234",
				Workflow: nebulav1.Workflow{
					Parameters: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
						"Hello": "World!",
					}),
					Name: "my-workflow",
					Steps: []*nebulav1.WorkflowStep{
						{
							Name:  "my-test-step",
							Image: "alpine:latest",
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
								"param": map[string]interface{}{
									"$type": "Parameter",
									"name":  "Hello",
								},
							}),
							Input: []string{
								"trap : TERM INT",
								"sleep 600 & wait",
							},
						},
					},
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wr))

		// Wait for step to start. Could use a ListWatcher but meh.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
				Namespace: wr.GetNamespace(),
				Name:      wr.GetName(),
			}, wr); err != nil {
				return retry.RetryPermanent(err)
			}

			if wr.Status.Steps["my-test-step"].Status == string(obj.WorkflowRunStatusInProgress) {
				return retry.RetryPermanent(nil)
			}

			return retry.RetryTransient(fmt.Errorf("waiting for step to start"))
		}))

		// Pull the pod and get its IP.
		pod := &corev1.Pod{}
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			pods := &corev1.PodList{}
			if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.InNamespace(cfg.Namespace.GetName()), client.MatchingLabels{
				// TODO: We shouldn't really hardcode this.
				"tekton.dev/task": (&model.Step{Run: model.Run{ID: wr.Spec.Name}, Name: "my-test-step"}).Hash().HexEncoding(),
			}); err != nil {
				return retry.RetryPermanent(err)
			}

			if len(pods.Items) == 0 {
				return retry.RetryTransient(fmt.Errorf("waiting for pod"))
			}

			pod = &pods.Items[0]
			if pod.Status.PodIP == "" {
				return retry.RetryTransient(fmt.Errorf("waiting for pod IP"))
			} else if pod.Status.Phase == corev1.PodPending {
				return retry.RetryTransient(fmt.Errorf("waiting for pod to start"))
			}

			return retry.RetryPermanent(nil)
		}))

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
			"param": "World!",
		}, result.Value.Data)
	})
}
