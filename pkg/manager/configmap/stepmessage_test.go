package configmap_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
	"github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestStepMessageManager(t *testing.T) {
	ctx := context.Background()

	step := &model.Step{
		Run:  model.Run{ID: "test"},
		Name: "test-step",
	}

	stepMessage := &model.StepMessage{
		ID:      uuid.NewString(),
		Details: errors.New("An error has occurred").Error(),
		ConditionEvaluationResult: &model.ConditionEvaluationResult{
			Expression: parse.Tree([]interface{}{
				testutil.JSONInvocation("equals", []interface{}{
					testutil.JSONParameter("param1"), "foobar",
				}),
				testutil.JSONInvocation("notEquals", []interface{}{
					testutil.JSONParameter("param2"), "barfoo",
				}),
			}),
		},
	}

	obj := &corev1.ConfigMap{}
	sm := configmap.NewStepMessageManager(step, configmap.NewLocalConfigMap(obj))

	err := sm.Set(ctx, stepMessage)
	require.NoError(t, err)

	sml, err := sm.List(ctx)
	require.NoError(t, err)
	require.Len(t, sml, 1)

	require.Equal(t, stepMessage.ID, sml[0].ID)
	require.Equal(t, stepMessage.Details, sml[0].Details)
	require.Equal(t, stepMessage.ConditionEvaluationResult.Expression, sml[0].ConditionEvaluationResult.Expression)
}
