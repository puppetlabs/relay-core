package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestKVConfigMap(t *testing.T) {
	ctx := context.Background()

	cm := configmap.NewLocalConfigMap(&corev1.ConfigMap{})
	kcm := configmap.NewKVConfigMap(cm)

	require.NoError(t, kcm.Set(ctx, "foo", "bar"))
	require.NoError(t, kcm.Set(ctx, "baz", "quux"))

	val, err := kcm.Get(ctx, "foo")
	require.NoError(t, err)
	require.Equal(t, "bar", val)

	val, err = kcm.Get(ctx, "baz")
	require.NoError(t, err)
	require.Equal(t, "quux", val)

	_, err = kcm.Get(ctx, "bar")
	require.Equal(t, model.ErrNotFound, err)
}
