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
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func stepTaskLabelValue(wr *nebulav1.WorkflowRun, stepName string) string {
	return app.ModelStepObjectKey(client.ObjectKeyFromObject(wr), &model.Step{Run: model.Run{ID: wr.Spec.Name}, Name: stepName}).Name
}

func waitForStepToStart(t *testing.T, ctx context.Context, cfg *Config, wr *nebulav1.WorkflowRun, stepName string) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{
			Namespace: wr.GetNamespace(),
			Name:      wr.GetName(),
		}, wr); err != nil {
			return true, err
		}

		if wr.Status.Steps[stepName].Status == string(obj.WorkflowRunStatusInProgress) {
			return true, nil
		}

		return false, fmt.Errorf("waiting for step %q to start", stepName)
	}))
}

func waitForStepToSucceed(t *testing.T, ctx context.Context, cfg *Config, wr *nebulav1.WorkflowRun, stepName string) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{
			Namespace: wr.GetNamespace(),
			Name:      wr.GetName(),
		}, wr); err != nil {
			return true, err
		}

		switch obj.WorkflowRunStatus(wr.Status.Steps[stepName].Status) {
		case obj.WorkflowRunStatusSuccess:
			return true, nil
		case obj.WorkflowRunStatusFailure:
			return true, fmt.Errorf("step failed")
		default:
			return false, fmt.Errorf("waiting for step to succeed")
		}
	}))
}

func waitForStepPodToComplete(t *testing.T, ctx context.Context, cfg *Config, wr *nebulav1.WorkflowRun, stepName string) *corev1.Pod {
	waitForStepToStart(t, ctx, cfg, wr, stepName)

	pod := &corev1.Pod{}
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(wr.GetNamespace()), client.MatchingLabels{
			"tekton.dev/task": stepTaskLabelValue(wr, stepName),
		}); err != nil {
			return true, err
		}

		if len(pods.Items) == 0 {
			return false, fmt.Errorf("waiting for step %q pod with label tekton.dev/task=%s", stepName, stepTaskLabelValue(wr, stepName))
		}

		pod = &pods.Items[0]
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			return false, fmt.Errorf("waiting for step %q pod to complete", stepName)
		}

		return true, nil
	}))

	return pod
}

func waitForStepPodIP(t *testing.T, ctx context.Context, cfg *Config, wr *nebulav1.WorkflowRun, stepName string) *corev1.Pod {
	waitForStepToStart(t, ctx, cfg, wr, stepName)

	pod := &corev1.Pod{}
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(wr.GetNamespace()), client.MatchingLabels{
			"tekton.dev/task": stepTaskLabelValue(wr, stepName),
		}); err != nil {
			return true, err
		}

		if len(pods.Items) == 0 {
			return false, fmt.Errorf("waiting for pod")
		}

		pod = &pods.Items[0]
		if pod.Status.PodIP == "" {
			return false, fmt.Errorf("waiting for pod IP")
		} else if pod.Status.Phase == corev1.PodPending {
			return false, fmt.Errorf("waiting for pod to start")
		}

		return true, nil
	}))

	return pod
}

func TestWorkflowRunWithTenantToolInjectionUsingInput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPI,
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
		ConfigWithVolumeClaimAdmission,
	}, func(cfg *Config) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
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

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		var ns corev1.Namespace
		require.Equal(t, cfg.Namespace.GetName(), tenant.Status.Namespace)
		require.NoError(t, cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: tenant.Status.Namespace}, &ns))

		value := relayv1beta1.AsUnstructured("World!")
		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Parameters: []*relayv1beta1.Parameter{
					{
						Name:  "Hello",
						Value: &value,
					},
				},
				Steps: []*relayv1beta1.Step{
					{
						Name: "my-test-step",
						Container: relayv1beta1.Container{
							Image: "alpine:latest",
							Input: []string{
								"ls -la " + model.ToolsMountPath,
							},
						},
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, w))

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
				WorkflowRef: corev1.LocalObjectReference{
					Name: "my-workflow",
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		pod := waitForStepPodToComplete(t, ctx, cfg, wr, "my-test-step")
		require.Equal(t, corev1.PodSucceeded, pod.Status.Phase)

		podLogOptions := &corev1.PodLogOptions{
			Container: "step-step",
		}

		logs := cfg.Environment.StaticClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions)
		podLogs, err := logs.Stream(ctx)
		require.NoError(t, err)
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		require.NoError(t, err)

		str := buf.String()

		require.Contains(t, str, model.EntrypointCommand)
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
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
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

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		var ns corev1.Namespace
		require.Equal(t, cfg.Namespace.GetName(), tenant.Status.Namespace)
		require.NoError(t, cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: tenant.Status.Namespace}, &ns))

		value := relayv1beta1.AsUnstructured("World!")
		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Parameters: []*relayv1beta1.Parameter{
					{
						Name:  "Hello",
						Value: &value,
					},
				},
				Steps: []*relayv1beta1.Step{
					{
						Name: "my-test-step",
						Container: relayv1beta1.Container{
							Image:   "alpine:latest",
							Command: "ls",
							Args:    []string{"-la", model.ToolsMountPath},
						},
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, w))

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
				WorkflowRef: corev1.LocalObjectReference{
					Name: "my-workflow",
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		pod := waitForStepPodToComplete(t, ctx, cfg, wr, "my-test-step")
		require.Equal(t, corev1.PodSucceeded, pod.Status.Phase)

		podLogOptions := &corev1.PodLogOptions{
			Container: "step-step",
		}

		logs := cfg.Environment.StaticClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions)
		podLogs, err := logs.Stream(ctx)
		require.NoError(t, err)
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		require.NoError(t, err)

		str := buf.String()

		require.Contains(t, str, model.EntrypointCommand)
	})
}

// TestWorkflowRun tests that an instance of the controller, when given a run to
// process, correctly sets up a Tekton pipeline and that the resulting pipeline
// should be able to access a metadata API service.
func TestWorkflowRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	data := struct {
		accessKeyID     string
		secretAccessKey string

		repository string
		tag        string
		version    int

		dryrun      bool
		backoff     int
		environment string
	}{
		accessKeyID:     uuid.NewString(),
		secretAccessKey: uuid.NewString(),
		repository:      uuid.NewString(),
		tag:             uuid.NewString(),
		version:         2,
		dryrun:          true,
		backoff:         300,
		environment:     uuid.NewString(),
	}

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPI,
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
		// Set a secret and connection for this workflow to look up.
		cfg.Vault.SetSecret(t, "my-tenant-id", "foo", "Hello")
		cfg.Vault.SetSecret(t, "my-tenant-id", "accessKeyId", data.accessKeyID)
		cfg.Vault.SetSecret(t, "my-tenant-id", "secretAccessKey", data.secretAccessKey)
		cfg.Vault.SetConnection(t, "my-domain-id", "aws", "test", map[string]string{
			"accessKeyID":     data.accessKeyID,
			"secretAccessKey": data.secretAccessKey,
		})

		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{},
			},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		value1 := relayv1beta1.AsUnstructured(data.repository)
		value2 := relayv1beta1.AsUnstructured("latest")
		value3 := relayv1beta1.AsUnstructured(1)
		value4 := relayv1beta1.AsUnstructured(data.dryrun)

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Parameters: []*relayv1beta1.Parameter{
					{
						Name:  "repository",
						Value: &value1,
					},
					{
						Name:  "tag",
						Value: &value2,
					},
					{
						Name:  "version",
						Value: &value3,
					},
					{
						Name:  "dryrun",
						Value: &value4,
					},
					{
						Name:  "payload",
						Value: nil,
					},
				},
				Steps: []*relayv1beta1.Step{
					{
						Name: "my-test-step",
						Container: relayv1beta1.Container{
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
								"parameter1": map[string]interface{}{
									"$type": "Parameter",
									"name":  "repository",
								},
								"parameter2": map[string]interface{}{
									"$type": "Parameter",
									"name":  "tag",
								},
								"parameter3": map[string]interface{}{
									"$type": "Parameter",
									"name":  "version",
								},
								"parameter4": map[string]interface{}{
									"$type": "Parameter",
									"name":  "dryrun",
								},
								"parameter5": map[string]interface{}{
									"$type": "Parameter",
									"name":  "payload",
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
								"ENVIRONMENT": data.environment,
								"DRYRUN":      data.dryrun,
								"BACKOFF":     data.backoff,
							}),
							Input: []string{
								"trap : TERM INT",
								"sleep 600 & wait",
							},
						},
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, w))

		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-run",
				Namespace: cfg.Namespace.GetName(),
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
				Parameters: relayv1beta1.UnstructuredObject{
					"tag":     relayv1beta1.AsUnstructured(data.tag),
					"version": relayv1beta1.AsUnstructured(data.version),
				},
				WorkflowRef: corev1.LocalObjectReference{
					Name: "my-workflow",
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		pod := waitForStepPodIP(t, ctx, cfg, wr, "my-test-step")

		var result exprmodel.JSONResultEnvelope
		evaluateRequest := func(url string) exprmodel.JSONResultEnvelope {
			req, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)
			req.Header.Set("X-Forwarded-For", pod.Status.PodIP)

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
			assert.True(t, result.Complete)

			return result
		}

		specUrl := func() string {
			return fmt.Sprintf("%s/spec", cfg.MetadataAPIURL)
		}

		envUrl := func() string {
			return fmt.Sprintf("%s/environment", cfg.MetadataAPIURL)
		}

		envNameUrl := func(environmentName string) string {
			return fmt.Sprintf("%s/environment/%s", cfg.MetadataAPIURL, environmentName)
		}

		result = evaluateRequest(specUrl())
		assert.Equal(t, map[string]interface{}{
			"secret": "Hello",
			"connection": map[string]interface{}{
				"accessKeyID":     data.accessKeyID,
				"secretAccessKey": data.secretAccessKey,
			},
			"parameter1": data.repository,
			"parameter2": data.tag,
			"parameter3": float64(data.version),
			"parameter4": data.dryrun,
			"parameter5": nil,
		}, result.Value.Data)

		result = evaluateRequest(envUrl())
		assert.Equal(t, map[string]interface{}{
			"AWS_ACCESS_KEY_ID":     data.accessKeyID,
			"AWS_SECRET_ACCESS_KEY": data.secretAccessKey,
			"ENVIRONMENT":           data.environment,
			"DRYRUN":                data.dryrun,
			"BACKOFF":               float64(data.backoff),
		}, result.Value.Data)

		result = evaluateRequest(envNameUrl("AWS_ACCESS_KEY_ID"))
		assert.Equal(t, data.accessKeyID, result.Value.Data)

		result = evaluateRequest(envNameUrl("AWS_SECRET_ACCESS_KEY"))
		assert.Equal(t, data.secretAccessKey, result.Value.Data)

		result = evaluateRequest(envNameUrl("ENVIRONMENT"))
		assert.Equal(t, data.environment, result.Value.Data)

		result = evaluateRequest(envNameUrl("DRYRUN"))
		assert.Equal(t, data.dryrun, result.Value.Data)

		result = evaluateRequest(envNameUrl("BACKOFF"))
		assert.Equal(t, float64(data.backoff), result.Value.Data)
	})
}

func TestWorkflowRunWithoutWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-run",
				Namespace: cfg.Namespace.GetName(),
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
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		require.Error(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: wr.GetName(), Namespace: wr.GetNamespace()}, wr); err != nil {
				if k8serrors.IsNotFound(err) {
					return retry.Repeat(fmt.Errorf("waiting for initial workflow run"))
				}

				return true, err
			}

			if wr.Status.Status == "" {
				return retry.Repeat(fmt.Errorf("waiting for workflow run status"))
			}

			return true, nil
		}))
	})
}

func TestWorkflowRunWithInvalidWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-run",
				Namespace: cfg.Namespace.GetName(),
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
				WorkflowRef: corev1.LocalObjectReference{
					Name: "my-workflow",
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		require.Error(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: wr.GetName(), Namespace: wr.GetNamespace()}, wr); err != nil {
				if k8serrors.IsNotFound(err) {
					return retry.Repeat(fmt.Errorf("waiting for initial workflow run"))
				}

				return true, err
			}

			if wr.Status.Status == "" {
				return retry.Repeat(fmt.Errorf("waiting for workflow run status"))
			}

			return true, nil
		}))
	})
}

func TestWorkflowRunWithoutSteps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{},
			},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Steps: []*relayv1beta1.Step{},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, w))

		wr := &nebulav1.WorkflowRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-test-run",
				Namespace: cfg.Namespace.GetName(),
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
				WorkflowRef: corev1.LocalObjectReference{
					Name: "my-workflow",
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: wr.GetName(), Namespace: wr.GetNamespace()}, wr); err != nil {
				if k8serrors.IsNotFound(err) {
					return false, fmt.Errorf("waiting for initial workflow run")
				}

				return true, err
			}

			if wr.Status.Status == "" {
				return false, fmt.Errorf("waiting for workflow run status")
			}

			return true, nil
		}))

		require.Equal(t, string(obj.WorkflowRunStatusSuccess), wr.Status.Status)
		require.NotNil(t, wr.Status.StartTime)
		require.NotNil(t, wr.Status.CompletionTime)
	})
}

func TestWorkflowRunStepInitTime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPIBoundInCluster,
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
	}, func(cfg *Config) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{},
			},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Steps: []*relayv1beta1.Step{
					{
						// TODO: Once we have the entrypointer image in
						// test, we could just end-to-end test it here.
						Name: "my-test-step",
						Container: relayv1beta1.Container{
							Image: "alpine:latest",
							Input: []string{
								"apk --no-cache add curl",
								fmt.Sprintf(`curl -XPUT "${METADATA_API_URL}/timers/%s"`, model.TimerStepInit),
							},
						},
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, w))

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
				WorkflowRef: corev1.LocalObjectReference{
					Name: "my-workflow",
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

		waitForStepToSucceed(t, ctx, cfg, wr, "my-test-step")
		require.NotEmpty(t, wr.Status.Steps["my-test-step"].InitTime)
	})
}

func TestWorkflowRunInGVisor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPIBoundInCluster,
		ConfigWithTenantReconciler,
		ConfigWithWorkflowRunReconciler,
		ConfigWithPodEnforcementAdmission,
	}, func(cfg *Config) {
		if cfg.Environment.GVisorRuntimeClassName == "" {
			t.Skip("gVisor is not available on this platform")
		}

		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.New().String(),
			},
			Spec: relayv1beta1.TenantSpec{
				ToolInjection: relayv1beta1.ToolInjection{},
			},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		when := relayv1beta1.AsUnstructured(
			testutil.JSONInvocation("equals", []interface{}{
				testutil.JSONParameter("Hello"),
				"World!",
			}))

		tests := []struct {
			Name           string
			StepDefinition *relayv1beta1.Step
		}{
			{
				Name: "command",
				StepDefinition: &relayv1beta1.Step{
					Name: "my-test-step",
					Container: relayv1beta1.Container{
						Image:   "alpine:latest",
						Command: "dmesg",
					},
				},
			},
			{
				Name: "input",
				StepDefinition: &relayv1beta1.Step{
					Name: "my-test-step",
					Container: relayv1beta1.Container{
						Image: "alpine:latest",
						Input: []string{"dmesg"},
					},
				},
			},
			{
				Name: "command-with-condition",
				StepDefinition: &relayv1beta1.Step{
					Name: "my-test-step",
					Container: relayv1beta1.Container{
						Image:   "alpine:latest",
						Command: "dmesg",
					},
					When: &when,
				},
			},
		}
		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				workflowName := fmt.Sprintf("my-test-workflow-%s", test.Name)

				value := relayv1beta1.AsUnstructured("World!")
				w := &relayv1beta1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      workflowName,
						Namespace: cfg.Namespace.GetName(),
					},
					Spec: relayv1beta1.WorkflowSpec{
						Parameters: []*relayv1beta1.Parameter{
							{
								Name:  "Hello",
								Value: &value,
							},
						},
						Steps: []*relayv1beta1.Step{test.StepDefinition},
						TenantRef: corev1.LocalObjectReference{
							Name: tenant.GetName(),
						},
					},
				}
				require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, w))

				wr := &nebulav1.WorkflowRun{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: cfg.Namespace.GetName(),
						Name:      fmt.Sprintf("my-test-run-%s", test.Name),
					},
					Spec: nebulav1.WorkflowRunSpec{
						Name: "my-workflow-run-1234",
						WorkflowRef: corev1.LocalObjectReference{
							Name: workflowName,
						},
					},
				}
				require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, wr))

				waitForStepToSucceed(t, ctx, cfg, wr, "my-test-step")

				// Get the logs from the pod.
				pod := &corev1.Pod{}
				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					pods := &corev1.PodList{}
					if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(wr.GetNamespace()), client.MatchingLabels{
						"tekton.dev/task": stepTaskLabelValue(wr, "my-test-step"),
					}); err != nil {
						return true, err
					}

					if len(pods.Items) == 0 {
						return false, fmt.Errorf("waiting for pod")
					}

					pod = &pods.Items[0]
					return true, nil
				}))

				podLogOptions := &corev1.PodLogOptions{
					Container: "step-step",
				}

				logs := cfg.Environment.StaticClient.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, podLogOptions)
				podLogs, err := logs.Stream(ctx)
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
