package configmap_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestStepMetadataManager(t *testing.T) {
	ctx := context.Background()

	step := &model.Step{
		Run:  model.Run{ID: "foo"},
		Name: "bar",
	}

	stepMetadata := &model.StepMetadata{
		Image: "alpine:latest",
	}

	obj := &corev1.ConfigMap{}
	lcm := configmap.NewLocalConfigMap(obj)

	smm := configmap.NewStepMetadataManager(step, lcm)

	require.NoError(t, smm.Set(ctx, stepMetadata))

	resultMetadata, err := smm.Get(ctx)
	require.NoError(t, err)

	require.Equal(t, stepMetadata.Image, resultMetadata.Image)
}
