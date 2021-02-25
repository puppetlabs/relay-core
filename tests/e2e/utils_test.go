package e2e_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForTenant(t *testing.T, ctx context.Context, tn *relayv1beta1.Tenant) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := e2e.ControllerRuntimeClient.Get(ctx, client.ObjectKey{
			Namespace: tn.GetNamespace(),
			Name:      tn.GetName(),
		}, tn); err != nil {
			return true, err
		}

		if tn.GetGeneration() != tn.Status.ObservedGeneration {
			return false, fmt.Errorf("waiting for tenant to reconcile: generation mismatch")
		}

		for _, cond := range tn.Status.Conditions {
			if cond.Type != relayv1beta1.TenantReady {
				continue
			} else if cond.Status != corev1.ConditionTrue {
				break
			}

			return true, nil
		}

		return false, fmt.Errorf("waiting for tenant to reconcile: not ready")
	}))
}

func CreateAndWaitForTenant(t *testing.T, ctx context.Context, tn *relayv1beta1.Tenant) {
	require.NoError(t, e2e.ControllerRuntimeClient.Create(ctx, tn))
	WaitForTenant(t, ctx, tn)
}

func Mutate(t *testing.T, ctx context.Context, obj runtime.Object, fn func()) {
	key, err := client.ObjectKeyFromObject(obj)
	require.NoError(t, err)

	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		// Mutation function.
		fn()

		if err := e2e.ControllerRuntimeClient.Update(ctx, obj); errors.IsConflict(err) {
			// Controller changed object, reload.
			if err := e2e.ControllerRuntimeClient.Get(ctx, key, obj); err != nil {
				return true, err
			}

			return false, err
		} else if err != nil {
			return true, err
		}

		return true, nil
	}))
}
