package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestStepDecoratorManager(t *testing.T) {
	ctx := context.Background()

	step := &model.Step{
		Run:  model.Run{ID: "test"},
		Name: "test-step",
	}

	obj := &corev1.ConfigMap{}
	dm := configmap.NewStepDecoratorManager(step, configmap.NewLocalConfigMap(obj))

	decoratorValues := map[string]interface{}{
		"name":        "test-decorator",
		"type":        string(v1beta1.DecoratorTypeLink),
		"description": "some link location",
		"uri":         "https://unit-testing.relay.sh/decorator-location",
	}

	err := dm.Set(ctx, decoratorValues)
	require.NoError(t, err)

	decoratorList, err := dm.List(ctx)
	require.NoError(t, err)
	require.Len(t, decoratorList, 1)
	dec := decoratorList[0]
	require.Equal(t, "test-decorator", dec.Name)
	require.Equal(t, step, dec.Step)

	decObj, ok := dec.Value.(v1beta1.Decorator)
	require.True(t, ok)

	require.Equal(t, "test-decorator", decObj.Name)
	require.NotNil(t, decObj.Link)
	require.Equal(t, &v1beta1.DecoratorLink{
		URI:         "https://unit-testing.relay.sh/decorator-location",
		Description: "some link location",
	}, decObj.Link)
}
