package e2e_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTenantFinalizer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		child := fmt.Sprintf("%s-child", ns.GetName())

		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: child,
					},
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, tenant))

		// Wait for namespace.
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.Get(ctx, client.ObjectKey{
				Namespace: tenant.GetNamespace(),
				Name:      tenant.GetName(),
			}, tenant); err != nil {
				return true, err
			}

			for _, cond := range tenant.Status.Conditions {
				if cond.Type == relayv1beta1.TenantNamespaceReady && cond.Status == corev1.ConditionTrue {
					return true, nil
				}
			}

			return false, fmt.Errorf("waiting for namespace to be ready")
		}))

		// Get child namespace.
		namespace := &corev1.Namespace{}
		require.NoError(t, eit.ControllerClient.Get(ctx, client.ObjectKey{Name: child}, namespace))

		// Delete tenant.
		require.NoError(t, eit.ControllerClient.Delete(ctx, tenant))

		// Get child namespace again, should be gone after delete.
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.Get(ctx, client.ObjectKey{Name: child}, namespace); errors.IsNotFound(err) {
				return true, nil
			} else if err != nil {
				return true, err
			}

			return false, fmt.Errorf("waiting for namespace to terminate")
		}))
	})
}

func TestTenantAPITriggerEventSinkMissingSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		// Create tenant with event sink pointing at nonexistent secret.
		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				TriggerEventSink: relayv1beta1.TriggerEventSink{
					API: &relayv1beta1.APITriggerEventSink{
						URL: "http://stub.example.com",
						TokenFrom: &relayv1beta1.APITokenSource{
							SecretKeyRef: &relayv1beta1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "xyz",
								},
								Key: "test",
							},
						},
					},
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, tenant))

		// Wait for tenant to reconcile.
		var cond relayv1beta1.TenantCondition
		require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
			if err := eit.ControllerClient.Get(ctx, client.ObjectKey{
				Namespace: tenant.GetNamespace(),
				Name:      tenant.GetName(),
			}, tenant); err != nil {
				return true, err
			}

			for _, cond = range tenant.Status.Conditions {
				if cond.Type == relayv1beta1.TenantEventSinkReady && cond.Status == corev1.ConditionFalse {
					return true, nil
				}
			}

			return false, fmt.Errorf("waiting for tenant to reconcile")
		}))
		assert.Equal(t, obj.TenantStatusReasonEventSinkNotConfigured, cond.Reason)
	})
}

func TestTenantAPITriggerEventSinkWithSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			StringData: map[string]string{
				"token": "test",
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, secret))

		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				TriggerEventSink: relayv1beta1.TriggerEventSink{
					API: &relayv1beta1.APITriggerEventSink{
						URL: "http://stub.example.com",
						TokenFrom: &relayv1beta1.APITokenSource{
							SecretKeyRef: &relayv1beta1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secret.GetName(),
								},
								Key: "token",
							},
						},
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tenant)
	})
}

func TestTenantAPITriggerEventSinkWithNamespaceAndSecret(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		child := fmt.Sprintf("%s-child", ns.GetName())

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			StringData: map[string]string{
				"token": "test",
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, secret))

		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "my-test-tenant",
			},
			Spec: relayv1beta1.TenantSpec{
				NamespaceTemplate: relayv1beta1.NamespaceTemplate{
					Metadata: metav1.ObjectMeta{
						Name: child,
					},
				},
				TriggerEventSink: relayv1beta1.TriggerEventSink{
					API: &relayv1beta1.APITriggerEventSink{
						URL: "http://stub.example.com",
						TokenFrom: &relayv1beta1.APITokenSource{
							SecretKeyRef: &relayv1beta1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: secret.GetName(),
								},
								Key: "token",
							},
						},
					},
				},
			},
		}
		CreateAndWaitForTenant(t, ctx, eit, tenant)
	})
}

func TestTenantNamespaceUpdate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		child1 := fmt.Sprintf("%s-child-1", ns.GetName())
		child2 := fmt.Sprintf("%s-child-2", ns.GetName())

		tenant := &relayv1beta1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
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
		CreateAndWaitForTenant(t, ctx, eit, tenant)

		// Child namespace should now exist.
		var ns1 corev1.Namespace
		require.Equal(t, child1, tenant.Status.Namespace)
		require.NoError(t, eit.ControllerClient.Get(ctx, client.ObjectKey{Name: child1}, &ns1))

		// Change namespace in tenant.
		Mutate(t, ctx, eit, tenant, func() {
			tenant.Spec.NamespaceTemplate.Metadata.Name = child2
		})
		WaitForTenant(t, ctx, eit, tenant)

		// First child namespace should now not exist or have deletion timestamp
		// set, second should exist.
		var ns2 corev1.Namespace
		require.Equal(t, child2, tenant.Status.Namespace)
		require.NoError(t, eit.ControllerClient.Get(ctx, client.ObjectKey{Name: child2}, &ns2))

		if err := eit.ControllerClient.Get(ctx, client.ObjectKey{Name: child1}, &ns1); err != nil {
			require.True(t, errors.IsNotFound(err))
		} else {
			require.NotEmpty(t, ns1.GetDeletionTimestamp())
		}
	})
}
