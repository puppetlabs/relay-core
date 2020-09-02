package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/util/retry"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
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

func WaitForTenant(t *testing.T, ctx context.Context, tn *relayv1beta1.Tenant) {
	require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
		if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
			Namespace: tn.GetNamespace(),
			Name:      tn.GetName(),
		}, tn); err != nil {
			return retry.RetryPermanent(err)
		}

		if tn.GetGeneration() != tn.Status.ObservedGeneration {
			return retry.RetryTransient(fmt.Errorf("waiting for tenant to reconcile: generation mismatch"))
		}

		for _, cond := range tn.Status.Conditions {
			if cond.Type != relayv1beta1.TenantReady {
				continue
			} else if cond.Status != corev1.ConditionTrue {
				break
			}

			return retry.RetryPermanent(nil)
		}

		return retry.RetryTransient(fmt.Errorf("waiting for tenant to reconcile: not ready"))
	}))
}

func CreateAndWaitForTenant(t *testing.T, ctx context.Context, tn *relayv1beta1.Tenant) {
	require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))
	WaitForTenant(t, ctx, tn)
}

func Mutate(t *testing.T, ctx context.Context, obj runtime.Object, fn func()) {
	key, err := client.ObjectKeyFromObject(obj)
	require.NoError(t, err)

	require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
		// Mutation function.
		fn()

		if err := e2e.ControllerRuntimeClient.Update(ctx, obj); errors.IsConflict(err) {
			// Controller changed object, reload.
			if err := e2e.ControllerRuntimeClient.Get(ctx, key, obj); err != nil {
				return retry.RetryPermanent(err)
			}

			return retry.RetryTransient(err)
		} else {
			return retry.RetryPermanent(err)
		}

		return retry.RetryTransient(nil)
	}))
}
