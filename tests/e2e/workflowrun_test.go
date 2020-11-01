package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/operator/obj"
	"github.com/puppetlabs/relay-core/pkg/util/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkflowRunWithTenantToolInjectionUsingInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPI,
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
		ConfigWithVolumeClaimAdmission,
	}, func(cfg *Config) {
		size, _ := resource.ParseQuantity("50Mi")
		storageClassName := "relay-hostpath"
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
			},
			Spec: relayv1beta1.TenantSpec{
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

		CreateAndWaitForTenant(t, ctx, tenant)

		var ns corev1.Namespace
		require.Equal(t, cfg.Namespace.GetName(), tenant.Status.Namespace)
		require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: tenant.Status.Namespace}, &ns))

		var pvc corev1.PersistentVolumeClaim
		require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: tenant.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tenant.Status.Namespace}, &pvc))

		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tenant.Status.Namespace,
				Name:      "my-test-run",
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/" + tenant.GetName(),
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            tenant.GetName(),
				},
			},
			Spec: nebulav1.WorkflowRunSpec{
				Name: "my-workflow-run-1234",
				TenantRef: &corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
				Workflow: nebulav1.Workflow{
					Parameters: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
						"Hello": "World!",
					}),
					Name: "my-workflow",
					Steps: []*nebulav1.WorkflowStep{
						{
							Name:  "my-test-step",
							Image: "alpine:latest",
							Input: []string{
								"ls -la " + model.ToolInjectionMountPath,
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

		pod := &corev1.Pod{}
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			pods := &corev1.PodList{}
			if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.InNamespace(tenant.Status.Namespace), client.MatchingLabels{
				"tekton.dev/task": (&model.Step{Run: model.Run{ID: wr.Spec.Name}, Name: "my-test-step"}).Hash().HexEncoding(),
			}); err != nil {
				return retry.RetryPermanent(err)
			}

			if len(pods.Items) == 0 {
				return retry.RetryTransient(fmt.Errorf("waiting for pod"))
			}

			pod = &pods.Items[0]
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				return retry.RetryTransient(fmt.Errorf("waiting for pod to complete"))
			}

			return retry.RetryPermanent(nil)
		}))
		require.Equal(t, corev1.PodSucceeded, pod.Status.Phase)

		podLogOptions := &corev1.PodLogOptions{
			Container: "step-step",
		}

		logs := e2e.Interface.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions)
		podLogs, err := logs.Stream()
		require.NoError(t, err)
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		require.NoError(t, err)

		str := buf.String()

		require.Contains(t, str, model.EntrypointCommand)

		e2e.ControllerRuntimeClient.Delete(ctx, &pvc)
	})
}

func TestWorkflowRunWithTenantToolInjectionUsingCommand(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPI,
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
		ConfigWithVolumeClaimAdmission,
	}, func(cfg *Config) {
		size, _ := resource.ParseQuantity("50Mi")
		storageClassName := "relay-hostpath"
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
			},
			Spec: relayv1beta1.TenantSpec{
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

		CreateAndWaitForTenant(t, ctx, tenant)

		var ns corev1.Namespace
		require.Equal(t, cfg.Namespace.GetName(), tenant.Status.Namespace)
		require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: tenant.Status.Namespace}, &ns))

		var pvc corev1.PersistentVolumeClaim
		require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: tenant.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tenant.Status.Namespace}, &pvc))

		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: tenant.Status.Namespace,
				Name:      "my-test-run",
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/" + tenant.GetName(),
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            tenant.GetName(),
				},
			},
			Spec: nebulav1.WorkflowRunSpec{
				Name: "my-workflow-run-1234",
				TenantRef: &corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
				Workflow: nebulav1.Workflow{
					Parameters: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
						"Hello": "World!",
					}),
					Name: "my-workflow",
					Steps: []*nebulav1.WorkflowStep{
						{
							Name:    "my-test-step",
							Image:   "alpine:latest",
							Command: "ls",
							Args:    []string{"-la", model.ToolInjectionMountPath},
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

		pod := &corev1.Pod{}
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			pods := &corev1.PodList{}
			if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.InNamespace(tenant.Status.Namespace), client.MatchingLabels{
				"tekton.dev/task": (&model.Step{Run: model.Run{ID: wr.Spec.Name}, Name: "my-test-step"}).Hash().HexEncoding(),
			}); err != nil {
				return retry.RetryPermanent(err)
			}

			if len(pods.Items) == 0 {
				return retry.RetryTransient(fmt.Errorf("waiting for pod"))
			}

			pod = &pods.Items[0]
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				return retry.RetryTransient(fmt.Errorf("waiting for pod to complete"))
			}

			return retry.RetryPermanent(nil)
		}))
		require.Equal(t, corev1.PodSucceeded, pod.Status.Phase)

		podLogOptions := &corev1.PodLogOptions{
			Container: "step-step",
		}

		logs := e2e.Interface.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions)
		podLogs, err := logs.Stream()
		require.NoError(t, err)
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		require.NoError(t, err)

		str := buf.String()

		require.Contains(t, str, model.EntrypointCommand)

		e2e.ControllerRuntimeClient.Delete(ctx, &pvc)
	})
}

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
		cfg.Vault.SetSecret(t, "my-tenant-id", "accessKeyId", "AKIA123456789")
		cfg.Vault.SetSecret(t, "my-tenant-id", "secretAccessKey", "that's-a-very-nice-key-you-have-there")
		cfg.Vault.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
			"accessKeyID":     "AKIA123456789",
			"secretAccessKey": "that's-a-very-nice-key-you-have-there",
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
							Env: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
								"AWS_ACCESS_KEY_ID": map[string]interface{}{
									"$type": "Secret",
									"name":  "accessKeyId",
								},
								"AWS_SECRET_ACCESS_KEY": map[string]interface{}{
									"$type": "Secret",
									"name":  "secretAccessKey",
								},
								"ENVIRONMENT": "test",
								"DRYRUN":      true,
								"BACKOFF":     300,
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
			if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.InNamespace(wr.GetNamespace()), client.MatchingLabels{
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
				"secretAccessKey": "that's-a-very-nice-key-you-have-there",
			},
			"param": "World!",
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
			"ENVIRONMENT":           "test",
			"DRYRUN":                true,
			"BACKOFF":               float64(300),
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

		req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/environment/ENVIRONMENT", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, "test", result.Value.Data)

		req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/environment/DRYRUN", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, true, result.Value.Data)

		req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("%s/environment/BACKOFF", cfg.MetadataAPIURL), nil)
		require.NoError(t, err)
		req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

		resp, err = http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
		assert.True(t, result.Complete)
		assert.Equal(t, float64(300), result.Value.Data)
	})
}

func TestWorkflowRunWithoutSteps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
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
					Name:  "my-workflow",
					Steps: []*nebulav1.WorkflowStep{},
				},
			},
		}
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wr))

		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: wr.GetName(), Namespace: wr.GetNamespace()}, wr); err != nil {
				if k8serrors.IsNotFound(err) {
					retry.RetryTransient(fmt.Errorf("waiting for initial workflow run"))
				}

				return retry.RetryPermanent(err)
			}

			if wr.Status.Status == "" {
				return retry.RetryTransient(fmt.Errorf("waiting for workflow run status"))
			}

			return retry.RetryPermanent(nil)
		}))

		require.Equal(t, string(obj.WorkflowRunStatusSuccess), wr.Status.Status)
		require.NotNil(t, wr.Status.StartTime)
		require.NotNil(t, wr.Status.CompletionTime)
	})
}

func TestWorkflowRunInGVisor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	if e2e.GVisorRuntimeClassName == "" {
		t.Skip("gVisor is not available on this platform")
	}

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPIBoundInCluster,
		ConfigWithWorkflowRunReconciler,
		ConfigWithPodEnforcementAdmission,
	}, func(cfg *Config) {
		tests := []struct {
			Name           string
			StepDefinition *nebulav1.WorkflowStep
		}{
			{
				Name: "command",
				StepDefinition: &nebulav1.WorkflowStep{
					Name:    "my-test-step",
					Image:   "alpine:latest",
					Command: "dmesg",
				},
			},
			{
				Name: "input",
				StepDefinition: &nebulav1.WorkflowStep{
					Name:  "my-test-step",
					Image: "alpine:latest",
					Input: []string{"dmesg"},
				},
			},
			{
				Name: "command-with-condition",
				StepDefinition: &nebulav1.WorkflowStep{
					Name:  "my-test-step",
					Image: "alpine:latest",
					When: relayv1beta1.AsUnstructured(
						testutil.JSONInvocation("equals", []interface{}{
							testutil.JSONParameter("Hello"),
							"World!",
						}),
					),
					Command: "dmesg",
				},
			},
		}
		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				wr := &nebulav1.WorkflowRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: cfg.Namespace.GetName(),
						Name:      fmt.Sprintf("my-test-run-%s", test.Name),
					},
					Spec: nebulav1.WorkflowRunSpec{
						Name: "my-workflow-run-1234",
						Workflow: nebulav1.Workflow{
							Name: "my-workflow",
							Parameters: relayv1beta1.NewUnstructuredObject(map[string]interface{}{
								"Hello": "World!",
							}),
							Steps: []*nebulav1.WorkflowStep{
								test.StepDefinition,
							},
						},
					},
				}
				require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, wr))

				// Wait for step to succeed.
				require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
					if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
						Namespace: wr.GetNamespace(),
						Name:      wr.GetName(),
					}, wr); err != nil {
						return retry.RetryPermanent(err)
					}

					switch obj.WorkflowRunStatus(wr.Status.Steps["my-test-step"].Status) {
					case obj.WorkflowRunStatusSuccess:
						return retry.RetryPermanent(nil)
					case obj.WorkflowRunStatusFailure:
						return retry.RetryPermanent(fmt.Errorf("step failed"))
					default:
						return retry.RetryTransient(fmt.Errorf("waiting for step to succeed"))
					}
				}))

				// Get the logs from the pod.
				pod := &corev1.Pod{}
				require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
					pods := &corev1.PodList{}
					if err := e2e.ControllerRuntimeClient.List(ctx, pods, client.InNamespace(wr.GetNamespace()), client.MatchingLabels{
						// TODO: We shouldn't really hardcode this.
						"tekton.dev/task": (&model.Step{Run: model.Run{ID: wr.Spec.Name}, Name: "my-test-step"}).Hash().HexEncoding(),
					}); err != nil {
						return retry.RetryPermanent(err)
					}

					if len(pods.Items) == 0 {
						return retry.RetryTransient(fmt.Errorf("waiting for pod"))
					}

					pod = &pods.Items[0]
					return retry.RetryPermanent(nil)
				}))

				podLogOptions := &corev1.PodLogOptions{
					Container: "step-step",
				}

				logs := e2e.Interface.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions)
				podLogs, err := logs.Stream()
				require.NoError(t, err)
				defer podLogs.Close()

				buf := new(bytes.Buffer)
				_, err = io.Copy(buf, podLogs)
				require.NoError(t, err)
				require.Contains(t, buf.String(), "gVisor")
			})
		}
	})
}
