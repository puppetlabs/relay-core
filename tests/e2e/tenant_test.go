package e2e_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	relayv1beta1 "github.com/puppetlabs/nebula-tasks/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/nebula-tasks/pkg/config"
	"github.com/puppetlabs/nebula-tasks/pkg/controller/tenant"
	"github.com/puppetlabs/nebula-tasks/pkg/util/retry"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestTenantFinalizer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	e2e.WithTestNamespace(t, ctx, func(ns *corev1.Namespace) {
		child := fmt.Sprintf("%s-child", ns.GetName())

		mgr, err := ctrl.NewManager(e2e.RESTConfig, ctrl.Options{
			Scheme: testutil.TestScheme,
		})
		require.NoError(t, err)
		require.NoError(t, tenant.Add(mgr, &config.WorkflowControllerConfig{
			Namespace: ns.GetName(),
		}))

		var wg sync.WaitGroup

		ch := make(chan struct{})

		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, mgr.Start(ch))
		}()
		defer func() {
			close(ch)
			wg.Wait()
		}()

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
		require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tenant))

		// Wait for namespace.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
				Namespace: tenant.GetNamespace(),
				Name:      tenant.GetName(),
			}, tenant); err != nil {
				return retry.RetryPermanent(err)
			}

			for _, cond := range tenant.Status.Conditions {
				if cond.Type == relayv1beta1.TenantNamespaceReady && cond.Status == corev1.ConditionTrue {
					return retry.RetryPermanent(nil)
				}
			}

			return retry.RetryTransient(fmt.Errorf("waiting for namespace to be ready"))
		}))

		// Get child namespace.
		namespace := &corev1.Namespace{}
		require.NoError(t, e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: child}, namespace))

		// Delete tenant.
		require.NoError(t, e2e.ControllerRuntimeClient.Delete(ctx, tenant))

		// Get child namespace again, should be gone after delete.
		require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
			if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{Name: child}, namespace); errors.IsNotFound(err) {
				return retry.RetryPermanent(nil)
			} else if err != nil {
				return retry.RetryPermanent(err)
			}

			return retry.RetryTransient(fmt.Errorf("waiting for namespace to terminate"))
		}))
	})
}
