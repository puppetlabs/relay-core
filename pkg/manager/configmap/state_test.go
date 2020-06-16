package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestStateManager(t *testing.T) {
	ctx := context.Background()

	step1 := &model.Step{
		Run:  model.Run{ID: "foo"},
		Name: "bar",
	}
	step2 := &model.Step{
		Run:  model.Run{ID: "foo"},
		Name: "baz",
	}

	obj := &corev1.ConfigMap{}
	sm1 := configmap.NewStateManager(step1, configmap.NewLocalConfigMap(obj))
	sm2 := configmap.NewStateManager(step2, configmap.NewLocalConfigMap(obj))

	_, err := sm1.Set(ctx, "key-a", "value-a-step-1")
	require.NoError(t, err)

	_, err = sm2.Set(ctx, "key-a", "value-a-step-2")
	require.NoError(t, err)

	_, err = sm1.Set(ctx, "key-b", "value-b-step-1")
	require.NoError(t, err)

	val, err := sm1.Get(ctx, "key-a")
	require.NoError(t, err)
	require.Equal(t, "value-a-step-1", val.Value)

	val, err = sm2.Get(ctx, "key-a")
	require.NoError(t, err)
	require.Equal(t, "value-a-step-2", val.Value)

	val, err = sm1.Get(ctx, "key-b")
	require.NoError(t, err)
	require.Equal(t, "value-b-step-1", val.Value)

	val, err = sm2.Get(ctx, "key-b")
	require.Equal(t, model.ErrNotFound, err)
}
