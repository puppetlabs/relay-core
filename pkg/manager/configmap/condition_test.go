package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/manager/configmap"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestConditionManager(t *testing.T) {
	ctx := context.Background()
	step := &model.Step{
		Run:  model.Run{ID: "foo"},
		Name: "bar",
	}

	cm := configmap.NewConditionManager(step, configmap.NewLocalConfigMap(&corev1.ConfigMap{}))

	cond, err := cm.Set(ctx, true)
	require.NoError(t, err)
	require.Equal(t, true, cond.Tree)

	cond, err = cm.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, true, cond.Tree)
}
