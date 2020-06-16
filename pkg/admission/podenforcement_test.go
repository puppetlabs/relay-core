package admission_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/admission"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type testServerInjectorHandler struct {
	http.Handler
}

var _ inject.Injector = testServerInjectorHandler{}

func (ts testServerInjectorHandler) InjectFunc(f inject.Func) error {
	if err := f(ts.Handler); err != nil {
		return err
	}

	inject.LoggerInto(log.Log.WithName("webhooks"), ts.Handler)
	return nil
}

func TestPodEnforcementHandler(t *testing.T) {
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

	hnd := testServerInjectorHandler{&webhook.Admission{Handler: admission.NewPodEnforcementHandler()}}
	mgr.SetFields(hnd)

	s := httptest.NewServer(hnd)
	defer s.Close()

	e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
		testutil.WithServiceBoundToHostHTTP(t, ctx, e2e.RESTConfig, e2e.Interface, s.URL, metav1.ObjectMeta{Namespace: ns.GetName()}, func(caPEM []byte, svc *corev1.Service) {
			// Set up webhook configuration in API server.
			cfg := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
				TypeMeta: metav1.TypeMeta{
					// Required for conversion during install, below.
					APIVersion: admissionregistrationv1beta1.SchemeGroupVersion.Identifier(),
					Kind:       "MutatingWebhookConfiguration",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-enforcement",
				},
				Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
					{
						Name: "pod-enforcement.admission.controller.relay.sh",
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
								"testing.relay.sh/pod-enforcement": "true",
							},
						},
					},
				},
			}
			// Patch instead of Create because this object is cluster-scoped
			// so we want to overwrite previous test attempts.
			require.NoError(t, e2e.ControllerRuntimeClient.Patch(ctx, cfg, client.Apply, client.ForceOwnership, client.FieldOwner("relay-e2e")))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, cfg))
			}()

			// Create child namespace to enforce rules on.
			child := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("%s-child", ns.GetName()),
					Labels: map[string]string{
						"testing.relay.sh/pod-enforcement": "true",
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, child))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, child))
			}()

			// Create a pod in the child namespace and check that the
			// admission controller applied our desired updates.
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: child.GetName(),
					Name:      "sneaky-pod",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "sneak",
							Image:   "alpine:latest",
							Command: []string{"sh", "-c", "trap : TERM INT; sleep 600 & wait"},
						},
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, pod))

			assert.Equal(t, admission.PodNodeSelector, pod.Spec.NodeSelector)
			assert.Equal(t, admission.PodTolerations, pod.Spec.Tolerations)
			assert.Equal(t, admission.PodDNSPolicy, pod.Spec.DNSPolicy)
			assert.Equal(t, admission.PodDNSConfig, pod.Spec.DNSConfig)
			// assert.Equal(t, admission.PodRuntimeClassName, pod.Spec.RuntimeClassName)
		})
	})
}
