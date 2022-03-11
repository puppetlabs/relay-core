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
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func stepTaskLabelValue(r *relayv1beta1.Run, stepName string) string {
	return app.ModelStepObjectKey(client.ObjectKeyFromObject(r), &model.Step{Run: model.Run{ID: r.GetName()}, Name: stepName}).Name
}

func waitForStepToStart(t *testing.T, ctx context.Context, cfg *Config, r *relayv1beta1.Run, stepName string) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{
			Namespace: r.GetNamespace(),
			Name:      r.GetName(),
		}, r); err != nil {
			return true, err
		}

		for _, step := range r.Status.Steps {
			if step.Name == stepName {
				for _, cond := range step.Conditions {
					if cond.Type == relayv1beta1.StepCompleted &&
						cond.Status != corev1.ConditionUnknown {
						return true, nil
					}
				}
			}
		}

		return false, fmt.Errorf("waiting for step %q to start", stepName)
	}))
}

func waitForStepToSucceed(t *testing.T, ctx context.Context, cfg *Config, r *relayv1beta1.Run, stepName string) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{
			Namespace: r.GetNamespace(),
			Name:      r.GetName(),
		}, r); err != nil {
			return true, err
		}

		for _, step := range r.Status.Steps {
			if step.Name == stepName {
				for _, cond := range step.Conditions {
					if cond.Type == relayv1beta1.StepSucceeded {
						switch cond.Status {
						case corev1.ConditionTrue:
							return true, nil
						case corev1.ConditionFalse:
							return true, fmt.Errorf("step failed")
						}
					}
				}
			}
		}

		return false, fmt.Errorf("waiting for step to succeed")
	}))
}

func waitForStepPodToComplete(t *testing.T, ctx context.Context, cfg *Config, r *relayv1beta1.Run, stepName string) *corev1.Pod {
	waitForStepToStart(t, ctx, cfg, r, stepName)

	pod := &corev1.Pod{}
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(r.GetNamespace()), client.MatchingLabels{
			"tekton.dev/task": stepTaskLabelValue(r, stepName),
		}); err != nil {
			return true, err
		}

		if len(pods.Items) == 0 {
			return false, fmt.Errorf("waiting for step %q pod with label tekton.dev/task=%s", stepName, stepTaskLabelValue(r, stepName))
		}

		pod = &pods.Items[0]
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			return false, fmt.Errorf("waiting for step %q pod to complete", stepName)
		}

		return true, nil
	}))

	return pod
}

func waitForStepPodIP(t *testing.T, ctx context.Context, cfg *Config, r *relayv1beta1.Run, stepName string) *corev1.Pod {
	waitForStepToStart(t, ctx, cfg, r, stepName)

	pod := &corev1.Pod{}
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(r.GetNamespace()), client.MatchingLabels{
			"tekton.dev/task": stepTaskLabelValue(r, stepName),
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

// TestRun tests that an instance of the controller, when given a run to
// process, correctly sets up a Tekton pipeline and that the resulting pipeline
// should be able to access a metadata API service.
func TestRun(t *testing.T) {
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
		ConfigWithRunReconciler,
	}, func(cfg *Config) {
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
				Name:      "tenant-" + uuid.NewString(),
			},
			Spec: relayv1beta1.TenantSpec{},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		value1 := relayv1beta1.AsUnstructured(data.repository)
		value2 := relayv1beta1.AsUnstructured("latest")
		value3 := relayv1beta1.AsUnstructured(1)
		value4 := relayv1beta1.AsUnstructured(data.dryrun)

		stepName := uuid.NewString()

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
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
						Name: stepName,
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

		r := &relayv1beta1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: cfg.Namespace.GetName(),
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            "my-tenant-id",
				},
			},
			Spec: relayv1beta1.RunSpec{
				Parameters: relayv1beta1.UnstructuredObject{
					"tag":     relayv1beta1.AsUnstructured(data.tag),
					"version": relayv1beta1.AsUnstructured(data.version),
				},
				WorkflowRef: corev1.LocalObjectReference{
					Name: w.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, r))

		pod := waitForStepPodIP(t, ctx, cfg, r, stepName)

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

func TestInvalidRuns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithRunReconciler,
	}, func(cfg *Config) {
		tests := []struct {
			Name string
			Run  *relayv1beta1.Run
		}{
			{
				Name: "missing-workflow-reference",
				Run: &relayv1beta1.Run{
					ObjectMeta: metav1.ObjectMeta{
						Name:      uuid.NewString(),
						Namespace: cfg.Namespace.GetName(),
						Annotations: map[string]string{
							model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
							model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
							model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
							model.RelayDomainIDAnnotation:            "my-domain-id",
							model.RelayTenantIDAnnotation:            "my-tenant-id",
						},
					},
					Spec: relayv1beta1.RunSpec{},
				},
			},
			{
				Name: "invalid-workflow-reference",
				Run: &relayv1beta1.Run{
					ObjectMeta: metav1.ObjectMeta{
						Name:      uuid.NewString(),
						Namespace: cfg.Namespace.GetName(),
						Annotations: map[string]string{
							model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
							model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
							model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
							model.RelayDomainIDAnnotation:            "my-domain-id",
							model.RelayTenantIDAnnotation:            "my-tenant-id",
						},
					},
					Spec: relayv1beta1.RunSpec{
						WorkflowRef: corev1.LocalObjectReference{
							Name: uuid.NewString(),
						},
					},
				},
			},
		}
		for _, test := range tests {
			t.Run(test.Name, func(t *testing.T) {
				r := test.Run
				require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, r))

				require.Error(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: r.GetName(), Namespace: r.GetNamespace()}, r); err != nil {
						if k8serrors.IsNotFound(err) {
							return retry.Repeat(fmt.Errorf("waiting for initial run"))
						}

						return retry.Done(err)
					}

					if len(r.Status.Conditions) > 0 {
						return retry.Done(nil)
					}

					return retry.Repeat(fmt.Errorf("waiting for run status"))
				}, retry.WithBackoffFactory(
					backoff.Build(
						backoff.Linear(100*time.Millisecond),
						backoff.MaxRetries(5),
					),
				)))
			})
		}
	})
}

func TestRunWithoutSteps(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithTenantReconciler,
		ConfigWithRunReconciler,
	}, func(cfg *Config) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.NewString(),
			},
			Spec: relayv1beta1.TenantSpec{},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
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

		r := &relayv1beta1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: cfg.Namespace.GetName(),
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            "my-tenant-id",
				},
			},
			Spec: relayv1beta1.RunSpec{
				WorkflowRef: corev1.LocalObjectReference{
					Name: w.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, r))

		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := cfg.Environment.ControllerClient.Get(ctx, client.ObjectKey{Name: r.GetName(), Namespace: r.GetNamespace()}, r); err != nil {
				if k8serrors.IsNotFound(err) {
					return retry.Repeat(fmt.Errorf("waiting for initial run"))
				}

				return retry.Done(err)
			}

			for _, cond := range r.Status.Conditions {
				if cond.Type == relayv1beta1.RunSucceeded {
					switch cond.Status {
					case corev1.ConditionTrue, corev1.ConditionFalse:
						return retry.Done(nil)
					}
				}
			}

			return retry.Repeat(fmt.Errorf("waiting for run status"))
		}))

		status := corev1.ConditionUnknown
		for _, cond := range r.Status.Conditions {
			if cond.Type == relayv1beta1.RunSucceeded {
				status = cond.Status
			}
		}

		require.Equal(t, corev1.ConditionTrue, status)
		require.NotNil(t, r.Status.StartTime)
		require.NotNil(t, r.Status.CompletionTime)
	})
}

func TestRunStepInitTime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPIBoundInCluster,
		ConfigWithTenantReconciler,
		ConfigWithRunReconciler,
	}, func(cfg *Config) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      "tenant-" + uuid.NewString(),
			},
			Spec: relayv1beta1.TenantSpec{},
		}

		CreateAndWaitForTenant(t, ctx, cfg, tenant)

		stepName := uuid.NewString()
		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: cfg.Namespace.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Steps: []*relayv1beta1.Step{
					{
						// TODO: Once we have the entrypointer image in
						// test, we could just end-to-end test it here.
						Name: stepName,
						Container: relayv1beta1.Container{
							Image: "alpine:latest",
							Input: []string{
								"apk --no-cache add curl",
								fmt.Sprintf(`curl -X PUT "${METADATA_API_URL}/timers/%s"`, model.TimerStepInit),
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

		r := &relayv1beta1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cfg.Namespace.GetName(),
				Name:      uuid.NewString(),
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    cfg.Vault.SecretsPath,
					model.RelayVaultConnectionPathAnnotation: "connections/my-domain-id",
					model.RelayVaultSecretPathAnnotation:     "workflows/my-tenant-id",
					model.RelayDomainIDAnnotation:            "my-domain-id",
					model.RelayTenantIDAnnotation:            "my-tenant-id",
				},
			},
			Spec: relayv1beta1.RunSpec{
				WorkflowRef: corev1.LocalObjectReference{
					Name: w.GetName(),
				},
			},
		}
		require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, r))

		waitForStepToSucceed(t, ctx, cfg, r, stepName)

		for _, step := range r.Status.Steps {
			if step.Name == stepName {
				require.NotEmpty(t, step.InitializationTime)
			}
		}
	})
}

func TestRunInGVisor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	WithConfig(t, ctx, []ConfigOption{
		ConfigWithMetadataAPIBoundInCluster,
		ConfigWithTenantReconciler,
		ConfigWithRunReconciler,
		ConfigWithPodEnforcementAdmission,
	}, func(cfg *Config) {
		if cfg.Environment.GVisorRuntimeClassName == "" {
			t.Skip("gVisor is not available on this platform")
		}

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
				tenant := &relayv1beta1.Tenant{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: cfg.Namespace.GetName(),
						Name:      "tenant-" + uuid.NewString(),
					},
					Spec: relayv1beta1.TenantSpec{},
				}

				CreateAndWaitForTenant(t, ctx, cfg, tenant)

				value := relayv1beta1.AsUnstructured("World!")
				w := &relayv1beta1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      uuid.NewString(),
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

				r := &relayv1beta1.Run{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: cfg.Namespace.GetName(),
						Name:      fmt.Sprintf("my-test-run-%s", test.Name),
					},
					Spec: relayv1beta1.RunSpec{
						WorkflowRef: corev1.LocalObjectReference{
							Name: w.GetName(),
						},
					},
				}
				require.NoError(t, cfg.Environment.ControllerClient.Create(ctx, r))

				waitForStepToSucceed(t, ctx, cfg, r, "my-test-step")

				pod := &corev1.Pod{}
				require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					pods := &corev1.PodList{}
					if err := cfg.Environment.ControllerClient.List(ctx, pods, client.InNamespace(r.GetNamespace()), client.MatchingLabels{
						"tekton.dev/task": stepTaskLabelValue(r, "my-test-step"),
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
