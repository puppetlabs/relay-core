package e2e_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func WaitForTenant(t *testing.T, ctx context.Context, eit *EnvironmentInTest, tn *relayv1beta1.Tenant) {
	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := eit.ControllerClient.Get(ctx, client.ObjectKey{
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

func CreateAndWaitForTenant(t *testing.T, ctx context.Context, eit *EnvironmentInTest, tn *relayv1beta1.Tenant) {
	require.NoError(t, eit.ControllerClient.Create(ctx, tn))
	WaitForTenant(t, ctx, eit, tn)
}

func Mutate(t *testing.T, ctx context.Context, eit *EnvironmentInTest, obj client.Object, fn func()) {
	key := client.ObjectKeyFromObject(obj)

	require.NoError(t, retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		// Mutation function.
		fn()

		if err := eit.ControllerClient.Update(ctx, obj); errors.IsConflict(err) {
			// Controller changed object, reload.
			if err := eit.ControllerClient.Get(ctx, key, obj); err != nil {
				return true, err
			}

			return false, err
		} else if err != nil {
			return true, err
		}

		return true, nil
	}))
}

func WaitForObjectDeletion(ctx context.Context, eit *EnvironmentInTest, obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)

	return retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := eit.ControllerClient.Get(ctx, key, obj); errors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return true, err
		}

		return false, fmt.Errorf("waiting for deletion of %T %s", obj, key)
	})
}

func CleanUp(t *testing.T, eit *EnvironmentInTest, ns *corev1.Namespace) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var del []client.Object

	wtl := &relayv1beta1.WebhookTriggerList{}
	require.NoError(t, eit.ControllerClient.List(ctx, wtl, client.InNamespace(ns.GetName())))
	if len(wtl.Items) > 0 {
		log.Printf("removing %d stale webhook trigger(s)", len(wtl.Items))
		for _, wt := range wtl.Items {
			func(wt relayv1beta1.WebhookTrigger) {
				del = append(del, &wt)
			}(wt)
		}
	}

	rl := &relayv1beta1.RunList{}
	require.NoError(t, eit.ControllerClient.List(ctx, rl, client.InNamespace(ns.GetName())))
	if len(rl.Items) > 0 {
		log.Printf("removing %d stale run(s)", len(rl.Items))
		for _, r := range rl.Items {
			func(r relayv1beta1.Run) {
				del = append(del, &r)
			}(r)
		}
	}

	wl := &relayv1beta1.WorkflowList{}
	require.NoError(t, eit.ControllerClient.List(ctx, wl, client.InNamespace(ns.GetName())))
	if len(wl.Items) > 0 {
		log.Printf("removing %d stale workflow(s)", len(wl.Items))
		for _, w := range wl.Items {
			func(w relayv1beta1.Workflow) {
				del = append(del, &w)
			}(w)
		}
	}

	tl := &relayv1beta1.TenantList{}
	require.NoError(t, eit.ControllerClient.List(ctx, tl, client.InNamespace(ns.GetName())))
	if len(tl.Items) > 0 {
		log.Printf("removing %d stale tenant(s)", len(tl.Items))
		for _, t := range tl.Items {
			func(t relayv1beta1.Tenant) {
				del = append(del, &t)
			}(t)
		}
	}

	for _, obj := range del {
		assert.NoError(t, eit.ControllerClient.Delete(ctx, obj))
		assert.NoError(t, WaitForObjectDeletion(ctx, eit, obj))
	}
}
