package v1

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowRunEngineMapping(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/valid.yaml")
	require.NoError(t, err)

	sd := NewDocumentStreamingDecoder(f, &YAMLDecoder{})

	wd, err := sd.DecodeStream(ctx)
	require.NoError(t, err)

	mapper := NewDefaultRunEngineMapper(
		WithNamespaceRunOption("valid-workflow"),
		WithWorkflowNameRunOption("valid-workflow-name"),
		WithWorkflowRunNameRunOption("valid-workflow-run-name"),
	)

	manifest, err := mapper.ToRuntimeObjectsManifest(wd)
	require.NoError(t, err)

	require.NotNil(t, manifest.Namespace)
	require.NotNil(t, manifest.WorkflowRun)

	require.Equal(t, "valid-workflow", manifest.Namespace.GetName())
	require.Equal(t, "valid-workflow-run-name", manifest.WorkflowRun.GetName())
	require.Equal(t, "valid-workflow-name", manifest.WorkflowRun.Spec.Workflow.Name)

	require.Len(t, manifest.WorkflowRun.Spec.Workflow.Steps, 1)
	require.Len(t, manifest.WorkflowRun.Spec.Workflow.Steps[0].Spec, 1)
	require.Len(t, manifest.WorkflowRun.Spec.Workflow.Steps[0].Env, 2)

	require.Len(t, manifest.WorkflowRun.Spec.Workflow.Parameters, 1)

	require.NoError(t, json.NewEncoder(ioutil.Discard).Encode(manifest.Namespace))
	require.NoError(t, json.NewEncoder(ioutil.Discard).Encode(manifest.WorkflowRun))
}
