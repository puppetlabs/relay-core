package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestActionStatusManager(t *testing.T) {
	ctx := context.Background()

	tcs := []struct {
		Name         string
		Action       model.Action
		ActionStatus *model.ActionStatus
	}{
		{
			Name: "Step action status",
			Action: &model.Step{
				Run:  model.Run{ID: "test"},
				Name: "test-step",
			},
			ActionStatus: &model.ActionStatus{
				ExitCode: 1,
			},
		},
		{
			Name: "Trigger action status",
			Action: &model.Trigger{
				Name: "test-trigger",
			},
			ActionStatus: &model.ActionStatus{
				ExitCode: 1,
			},
		},
	}

	for _, test := range tcs {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			obj := &corev1.ConfigMap{}
			am := configmap.NewActionStatusManager(test.Action, configmap.NewLocalConfigMap(obj))

			err := am.Set(ctx, test.ActionStatus)
			require.NoError(t, err)

			actual, err := am.Get(ctx, test.Action)
			require.NoError(t, err)

			switch test.Action.Type().Singular {
			case model.ActionTypeStep.Singular:
				require.Equal(t, test.ActionStatus, actual)
			case model.ActionTypeTrigger.Singular:
				require.Nil(t, actual)
			}
		})
	}
}
