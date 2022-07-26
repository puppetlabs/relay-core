package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestConditionManagerWhenConditionsAreDefined(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		Name                        string
		Action                      model.Action
		Value                       interface{}
		ExpectedConditionExpression any
	}{
		{
			Name: "Valid step and value",
			Action: &model.Step{
				Run:  model.Run{ID: "foo"},
				Name: "bar",
			},
			Value:                       true,
			ExpectedConditionExpression: true,
		},
		{
			Name: "Valid step and nil value",
			Action: &model.Step{
				Run:  model.Run{ID: "foo"},
				Name: "bar",
			},
			Value: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cm := configmap.NewConditionManager(test.Action, configmap.NewLocalConfigMap(&corev1.ConfigMap{}))

			condition, err := cm.Set(ctx, test.Value)
			require.NoError(t, err)
			require.NotNil(t, condition)
			require.Equal(t, test.ExpectedConditionExpression, condition.Tree)

			condition, err = cm.Get(ctx)

			require.NoError(t, err)
			require.NotNil(t, condition)
			require.Equal(t, test.ExpectedConditionExpression, condition.Tree)
		})
	}
}

func TestConditionManagerWhenNoConditionsAreDefined(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		Name          string
		Action        model.Action
		ExpectedError error
	}{
		{
			Name: "Valid step and no value set",
			Action: &model.Step{
				Run:  model.Run{ID: "foo"},
				Name: "bar",
			},
			ExpectedError: model.ErrNotFound,
		},
		{
			Name: "Valid trigger and no value set",
			Action: &model.Trigger{
				Name: "bar",
			},
			ExpectedError: model.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			cm := configmap.NewConditionManager(test.Action, configmap.NewLocalConfigMap(&corev1.ConfigMap{}))

			condition, err := cm.Get(ctx)

			if test.ExpectedError != nil {
				require.Equal(t, test.ExpectedError, err)
				require.Nil(t, condition)
			} else {
				require.NoError(t, err)
				require.NotNil(t, condition)
			}
		})
	}
}
