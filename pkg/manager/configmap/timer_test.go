package configmap_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestTimerManager(t *testing.T) {
	ctx := context.Background()
	step := &model.Step{
		Run:  model.Run{ID: "foo"},
		Name: "bar",
	}

	cm := configmap.NewTimerManager(step, configmap.NewLocalConfigMap(&corev1.ConfigMap{}))

	now := time.Now().Truncate(time.Second).In(time.UTC)

	timer, err := cm.Set(ctx, "test", now)
	require.NoError(t, err)
	require.Equal(t, "test", timer.Name)
	require.Equal(t, now, timer.Time)

	timer, err = cm.Get(ctx, "test")
	require.NoError(t, err)
	require.Equal(t, "test", timer.Name)
	require.Equal(t, now, timer.Time)

	timer, err = cm.Get(ctx, "test-2")
	require.Equal(t, model.ErrNotFound, err)

	timer, err = cm.Set(ctx, "test", time.Now())
	require.Equal(t, model.ErrConflict, err)

	timer, err = cm.Get(ctx, "test")
	require.NoError(t, err)
	require.Equal(t, "test", timer.Name)
	require.Equal(t, now, timer.Time)
}
