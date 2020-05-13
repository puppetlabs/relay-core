package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/manager/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func TestClientConfigMap(t *testing.T) {
	ctx := context.Background()
	kc := testutil.NewMockKubernetesClient()

	cm := configmap.NewClientConfigMap(kc, "default", "test")

	obj, err := cm.Get(ctx)
	require.Nil(t, obj)
	require.True(t, errors.IsNotFound(err))

	obj = &corev1.ConfigMap{
		Data: map[string]string{
			"foo": "bar",
		},
	}
	obj, err = cm.CreateOrUpdate(ctx, obj)
	require.NoError(t, err)
	require.NotEmpty(t, obj.GetUID())
	require.Equal(t, "default", obj.GetNamespace())
	require.Equal(t, "test", obj.GetName())
	require.Equal(t, "bar", obj.Data["foo"])

	obj.Data["foo"] = "baz"
	obj, err = cm.CreateOrUpdate(ctx, obj)
	require.NoError(t, err)
	require.Equal(t, "baz", obj.Data["foo"])
}

func TestLocalConfigMap(t *testing.T) {
	ctx := context.Background()
	obj := &corev1.ConfigMap{
		Data: map[string]string{
			"foo": "bar",
		},
	}

	cm := configmap.NewLocalConfigMap(obj)

	copy, err := cm.Get(ctx)
	require.NoError(t, err)

	// Make sure we're using a copy.
	copy.Data["foo"] = "baz"
	require.Equal(t, "bar", obj.Data["foo"])

	// Make sure that save propagates back to the original map.
	copy, err = cm.CreateOrUpdate(ctx, copy)
	require.NoError(t, err)
	require.Equal(t, "baz", obj.Data["foo"])
}

func TestMutateConfigMap(t *testing.T) {
	ctx := context.Background()
	kc := testutil.NewMockKubernetesClient()

	cm := configmap.NewClientConfigMap(kc, "default", "test")

	obj, err := configmap.MutateConfigMap(ctx, cm, func(obj *corev1.ConfigMap) {
		obj.Data["foo"] = "bar"
	})
	require.NoError(t, err)
	require.NotEmpty(t, obj.GetUID())
	require.Equal(t, "test", obj.GetName())
	require.Equal(t, "bar", obj.Data["foo"])

	obj, err = configmap.MutateConfigMap(ctx, cm, func(obj *corev1.ConfigMap) {
		obj.Data["foo"] = "baz"
	})
	require.NoError(t, err)
	require.Equal(t, "baz", obj.Data["foo"])
}
