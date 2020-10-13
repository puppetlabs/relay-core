package testutil

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/operator/admission"
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

func WithPodEnforcementAdmissionRegistration(t *testing.T, ctx context.Context, e2e *EndToEndEnvironment, mgr manager.Manager, opts []admission.PodEnforcementHandlerOption, namespaceSelector *metav1.LabelSelector, fn func()) {
	hnd := testServerInjectorHandler{&webhook.Admission{Handler: admission.NewPodEnforcementHandler(opts...)}}
	mgr.SetFields(hnd)

	s := httptest.NewServer(hnd)
	defer s.Close()

	if namespaceSelector == nil {
		namespaceSelector = e2e.TestNamespaceSelector(t)
	}

	hash := sha1.Sum([]byte(namespaceSelector.String()))
	name := fmt.Sprintf("pod-enforcement-%x.admission.controller.relay.sh", hash[:])

	e2e.WithUtilNamespace(t, ctx, func(ns *corev1.Namespace) {
		WithServiceBoundToHostHTTP(t, ctx, e2e.RESTConfig, e2e.Interface, s.URL, metav1.ObjectMeta{Namespace: ns.GetName()}, func(caPEM []byte, svc *corev1.Service) {
			// Set up webhook configuration in API server.
			cfg := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
				TypeMeta: metav1.TypeMeta{
					// Required for conversion during install, below.
					APIVersion: admissionregistrationv1beta1.SchemeGroupVersion.Identifier(),
					Kind:       "MutatingWebhookConfiguration",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
					{
						Name: name,
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
						NamespaceSelector: namespaceSelector,
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

			fn()
		})
	})
}

func WithVolumeClaimAdmissionRegistration(t *testing.T, ctx context.Context, e2e *EndToEndEnvironment, mgr manager.Manager, opts []admission.VolumeClaimHandlerOption, namespaceSelector *metav1.LabelSelector, fn func()) {
	hnd := testServerInjectorHandler{&webhook.Admission{Handler: admission.NewVolumeClaimHandler(opts...)}}
	mgr.SetFields(hnd)

	s := httptest.NewServer(hnd)
	defer s.Close()

	if namespaceSelector == nil {
		namespaceSelector = e2e.TestNamespaceSelector(t)
	}

	hash := sha1.Sum([]byte(namespaceSelector.String()))
	name := fmt.Sprintf("volume-claim-%x.admission.controller.relay.sh", hash[:])

	e2e.WithUtilNamespace(t, ctx, func(ns *corev1.Namespace) {
		WithServiceBoundToHostHTTP(t, ctx, e2e.RESTConfig, e2e.Interface, s.URL, metav1.ObjectMeta{Namespace: ns.GetName()}, func(caPEM []byte, svc *corev1.Service) {
			cfg := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
				TypeMeta: metav1.TypeMeta{
					APIVersion: admissionregistrationv1beta1.SchemeGroupVersion.Identifier(),
					Kind:       "MutatingWebhookConfiguration",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Webhooks: []admissionregistrationv1beta1.MutatingWebhook{
					{
						Name: name,
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
						NamespaceSelector: namespaceSelector,
					},
				},
			}
			require.NoError(t, e2e.ControllerRuntimeClient.Patch(ctx, cfg, client.Apply, client.ForceOwnership, client.FieldOwner("relay-e2e")))
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				assert.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, cfg))
			}()

			fn()
		})
	})
}
