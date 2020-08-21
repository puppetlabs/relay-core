package admission_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/admission"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

func TestVolumeClaimHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var e2e *testutil.EndToEndEnvironment
	testutil.WithEndToEndEnvironment(t, func(e *testutil.EndToEndEnvironment) {
		e2e = e
	})

	// Assume we're skipping this test if we do not have a valid environment
	// This does not currently have access to the enabled flag
	if e2e == nil {
		t.SkipNow()
	}

	mgr, err := manager.New(e2e.RESTConfig, manager.Options{
		Scheme:             testutil.TestScheme,
		MetricsBindAddress: "0",
	})
	require.NoError(t, err)

	hnd := testServerInjectorHandler{&webhook.Admission{Handler: admission.NewVolumeClaimHandler()}}
	mgr.SetFields(hnd)

	s := httptest.NewServer(hnd)
	defer s.Close()

	e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
		testutil.WithServiceBoundToHostHTTP(t, ctx, e2e.RESTConfig, e2e.Interface, s.URL, metav1.ObjectMeta{Namespace: ns.GetName()}, func(caPEM []byte, svc *corev1.Service) {
			cfg := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
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
			require.NoError(t, e2e.ControllerRuntimeClient.Patch(ctx, cfg, client.Apply, client.ForceOwnership, client.FieldOwner("relay-e2e")))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, cfg))
			}()

			child := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-child", ns.GetName()),
					Labels: map[string]string{
						"testing.relay.sh/tools-volume-claim": "true",
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, child))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, child))
			}()

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: child.GetName(),
					Name:      "sneaky-pod",
					Annotations: map[string]string{
						model.RelayControllerToolsVolumeClaimAnnotation: "tools-volume-claim",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "hide",
							Image:   "alpine:latest",
							Command: []string{"sh", "-c", "trap : TERM INT; sleep 600 & wait"},
						},
						{
							Name:    "sneak",
							Image:   "alpine:latest",
							Command: []string{"sh", "-c", "trap : TERM INT; sleep 600 & wait"},
						},
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, pod))

			volume := false
			for _, v := range pod.Spec.Volumes {
				if v.Name == model.ToolInjectionMountName &&
					v.VolumeSource.PersistentVolumeClaim.ClaimName == "tools-volume-claim" {
					volume = true
				}
			}

			assert.True(t, volume)

			volumeMount := false
			for _, c := range pod.Spec.Containers {
				for _, vm := range c.VolumeMounts {
					if vm.Name == model.ToolInjectionMountName {
						volumeMount = true
					}
				}

				assert.True(t, volumeMount)
			}
		})
	})
}
