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

	name := "test-decorator"
	typ := model.DecoratorTypeLink

	decoratorValues := map[string]interface{}{
		"description": "some link location",
		"uri":         "https://unit-testing.relay.sh/decorator-location",
	}

	err := dm.Set(ctx, string(typ), name, decoratorValues)
	require.NoError(t, err)

	decoratorList, err := dm.List(ctx)
	require.NoError(t, err)
	require.Len(t, decoratorList, 1)
	dec := decoratorList[0]
	require.Equal(t, name, dec.Name)
	require.Equal(t, step, dec.Step)

	require.Equal(t, name, dec.Value.Name)
	require.NotNil(t, dec.Value.Link)
	require.Equal(t, &v1beta1.DecoratorLink{
		URI:         "https://unit-testing.relay.sh/decorator-location",
		Description: "some link location",
	}, dec.Value.Link)
}
