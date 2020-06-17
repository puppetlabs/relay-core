package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestSpecManager(t *testing.T) {
	ctx := context.Background()
	step := &model.Step{
		Run:  model.Run{ID: "foo"},
		Name: "bar",
	}

	sm := configmap.NewSpecManager(step, configmap.NewLocalConfigMap(&corev1.ConfigMap{}))

	spec, err := sm.Set(ctx, map[string]interface{}{"foo": "bar"})
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"foo": "bar"}, spec.Tree)

	spec, err = sm.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"foo": "bar"}, spec.Tree)
}
