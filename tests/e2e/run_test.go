package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/puppetlabs/leg/k8sutil/pkg/app/exec"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/timeutil/pkg/backoff"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
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

func waitForStepToStart(t *testing.T, ctx context.Context, eit *EnvironmentInTest, r *relayv1beta1.Run, stepName string) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := eit.ControllerClient.Get(ctx, client.ObjectKey{
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

func waitForStepToSucceed(t *testing.T, ctx context.Context, eit *EnvironmentInTest, r *relayv1beta1.Run, stepName string) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := eit.ControllerClient.Get(ctx, client.ObjectKey{
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

func waitForStepPodIP(t *testing.T, ctx context.Context, eit *EnvironmentInTest, r *relayv1beta1.Run, stepName string) *corev1.Pod {
	waitForStepToStart(t, ctx, eit, r, stepName)

	pod := &corev1.Pod{}
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		if err := eit.ControllerClient.List(ctx, pods, client.InNamespace(r.GetNamespace()), client.MatchingLabels{
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

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "tenant-" + uuid.NewString(),
			},
			Spec: relayv1beta1.TenantSpec{},
		}

		WithVault(t, ctx, eit, func(v *Vault) {
			v.SetSecret(t, ctx, tenant.GetName(), "foo", "Hello")
			v.SetSecret(t, ctx, tenant.GetName(), "accessKeyId", data.accessKeyID)
			v.SetSecret(t, ctx, tenant.GetName(), "secretAccessKey", data.secretAccessKey)
			v.SetConnection(t, ctx, ns.GetName(), "aws", "test", map[string]string{
				"accessKeyID":     data.accessKeyID,
				"secretAccessKey": data.secretAccessKey,
			})
		})

		CreateAndWaitForTenant(t, ctx, eit, tenant)

		value1 := relayv1beta1.AsUnstructured(data.repository)
		value2 := relayv1beta1.AsUnstructured("latest")
		value3 := relayv1beta1.AsUnstructured(1)
		value4 := relayv1beta1.AsUnstructured(data.dryrun)

		stepName := uuid.NewString()

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: ns.GetName(),
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
		require.NoError(t, eit.ControllerClient.Create(ctx, w))

		r := &relayv1beta1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayVaultEngineMountAnnotation:    TestVaultEngineTenantPath,
					model.RelayVaultConnectionPathAnnotation: "connections/" + ns.GetName(),
					model.RelayVaultSecretPathAnnotation:     "workflows/" + tenant.GetName(),
					model.RelayDomainIDAnnotation:            ns.GetName(),
					model.RelayTenantIDAnnotation:            tenant.GetName(),
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
		require.NoError(t, eit.ControllerClient.Create(ctx, r))

		pod := waitForStepPodIP(t, ctx, eit, r, stepName)

		var result api.GetSpecResponseEnvelope
		evaluateRequest := func(url string) api.GetSpecResponseEnvelope {
			r, err := exec.ShellScript(ctx, eit.RESTConfig, corev1obj.NewPodFromObject(pod), fmt.Sprintf("exec wget -q -O - %s", url), exec.WithContainer(model.ActionPodStepContainerName))
			require.NoError(t, err)
			require.Equal(t, 0, r.ExitCode, "unexpected error from script: standard output:\n%s\n\nstandard error:\n%s", r.Stdout, r.Stderr)

			require.NoError(t, json.Unmarshal([]byte(r.Stdout), &result))
			assert.True(t, result.Complete)

			return result
		}

		specUrl := func() string {
			return fmt.Sprintf("$%s/spec", model.EnvironmentVariableMetadataAPIURL)
		}

		envUrl := func() string {
			return fmt.Sprintf("$%s/environment", model.EnvironmentVariableMetadataAPIURL)
		}

		envNameUrl := func(environmentName string) string {
			return fmt.Sprintf("$%s/environment/%s", model.EnvironmentVariableMetadataAPIURL, environmentName)
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

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tests := []struct {
			Name string
			Run  *relayv1beta1.Run
		}{
			{
				Name: "missing-workflow-reference",
				Run: &relayv1beta1.Run{
					ObjectMeta: metav1.ObjectMeta{
						Name:      uuid.NewString(),
						Namespace: ns.GetName(),
					},
					Spec: relayv1beta1.RunSpec{},
				},
			},
			{
				Name: "invalid-workflow-reference",
				Run: &relayv1beta1.Run{
					ObjectMeta: metav1.ObjectMeta{
						Name:      uuid.NewString(),
						Namespace: ns.GetName(),
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
				require.NoError(t, eit.ControllerClient.Create(ctx, r))

				require.Error(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
					if err := eit.ControllerClient.Get(ctx, client.ObjectKey{Name: r.GetName(), Namespace: r.GetNamespace()}, r); err != nil {
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

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "tenant-" + uuid.NewString(),
			},
			Spec: relayv1beta1.TenantSpec{},
		}

		CreateAndWaitForTenant(t, ctx, eit, tenant)

		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Steps: []*relayv1beta1.Step{},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, w))

		r := &relayv1beta1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: ns.GetName(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tenant.GetName(),
				},
			},
			Spec: relayv1beta1.RunSpec{
				WorkflowRef: corev1.LocalObjectReference{
					Name: w.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, r))

		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.Get(ctx, client.ObjectKey{Name: r.GetName(), Namespace: r.GetNamespace()}, r); err != nil {
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

func TestDependsOn(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "tenant-" + uuid.NewString(),
			},
			Spec: relayv1beta1.TenantSpec{},
		}

		CreateAndWaitForTenant(t, ctx, eit, tenant)

		step1 := uuid.NewString()
		step2 := uuid.NewString()
		w := &relayv1beta1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      uuid.NewString(),
				Namespace: ns.GetName(),
			},
			Spec: relayv1beta1.WorkflowSpec{
				Steps: []*relayv1beta1.Step{
					{
						Name: step1,
						Container: relayv1beta1.Container{
							Image: "alpine:latest",
							Input: []string{
								"exit 0",
							},
						},
					},
					{
						Name: step2,
						Container: relayv1beta1.Container{
							Image: "alpine:latest",
							Input: []string{
								"exit 0",
							},
						},
						DependsOn: []string{step1},
					},
				},
				TenantRef: corev1.LocalObjectReference{
					Name: tenant.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, w))

		r := &relayv1beta1.Run{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      uuid.NewString(),
				Annotations: map[string]string{
					model.RelayDomainIDAnnotation: ns.GetName(),
					model.RelayTenantIDAnnotation: tenant.GetName(),
				},
			},
			Spec: relayv1beta1.RunSpec{
				WorkflowRef: corev1.LocalObjectReference{
					Name: w.GetName(),
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, r))

		waitForStepToSucceed(t, ctx, eit, r, step2)

		for _, step := range r.Status.Steps {
			require.NotNil(t, step.CompletionTime)
		}
	})
}
