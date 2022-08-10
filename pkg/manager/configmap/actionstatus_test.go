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
				ProcessState: &model.ActionStatusProcessState{
					ExitCode: 1,
				},
			},
		},
		{
			Name: "Trigger action status",
			Action: &model.Trigger{
				Name: "test-trigger",
			},
			ActionStatus: &model.ActionStatus{
				ProcessState: &model.ActionStatusProcessState{
					ExitCode: 1,
				},
			},
		},
	}

	for _, test := range tcs {
		t.Run(test.Name, func(t *testing.T) {
			obj := &corev1.ConfigMap{}
			am := configmap.NewActionStatusManager(test.Action, configmap.NewLocalConfigMap(obj))

			err := am.Set(ctx, test.ActionStatus)
			require.NoError(t, err)

			actual, err := am.Get(ctx, test.Action)
			require.NoError(t, err)

			aml, err := am.List(ctx)
			require.NoError(t, err)

			switch at := test.Action.(type) {
			case *model.Step:
				expected := &model.ActionStatus{
					Name:          at.Name,
					ProcessState:  test.ActionStatus.ProcessState,
					WhenCondition: test.ActionStatus.WhenCondition,
				}
				require.Equal(t, expected, actual)
				require.Len(t, aml, 1)
			case *model.Trigger:
				require.Nil(t, actual)
				require.Len(t, aml, 0)
			default:
				require.Fail(t, "unexpected action type")
			}
		})
	}
}
