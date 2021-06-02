package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestParameterManager(t *testing.T) {
	ctx := context.Background()

	obj := &corev1.ConfigMap{}
	pm := configmap.NewParameterManager(configmap.NewLocalConfigMap(obj))

	_, err := pm.Set(ctx, "key-a", "value-a")
	require.NoError(t, err)
	_, err = pm.Set(ctx, "key-c", "value-c")
	require.NoError(t, err)

	val, err := pm.Get(ctx, "key-a")
	require.NoError(t, err)
	require.Equal(t, "key-a", val.Name)
	require.Equal(t, "value-a", val.Value)

	val, err = pm.Get(ctx, "key-b")
	require.Equal(t, model.ErrNotFound, err)

	vals, err := pm.List(ctx)
	require.Len(t, vals, 2)
	require.Contains(t, vals, &model.Parameter{Name: "key-a", Value: "value-a"})
	require.Contains(t, vals, &model.Parameter{Name: "key-c", Value: "value-c"})
}
