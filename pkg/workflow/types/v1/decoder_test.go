package v1

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func validWorkflow(t *testing.T, wd *WorkflowData) {
	expected := &WorkflowData{
		Version:     "v1",
		Description: "This is a workflow",
		Parameters: WorkflowParameters(map[string]*WorkflowParameter{
			"hi": {
				Default:     5,
				Description: "Hello",
			},
		}),
		Steps: []*WorkflowStep{
			{
				Name: "step-1",
				Variant: &ContainerWorkflowStep{
					ContainerMixin: ContainerMixin{
						Image: "image-1",
					},
				},
			},
		},
	}

	require.Equal(t, expected, wd)
}

func complicatedWorkflow(t *testing.T, wd *WorkflowData) {
	require.Equal(t, "v1", wd.Version)
	require.Equal(t, "a more complicated workflow", wd.Description)

	require.Len(t, wd.Parameters, 2)
	require.Equal(t, "param-1-default", wd.Parameters["param-1"].Default)
	require.Equal(t, "param-2-default", wd.Parameters["param-2"].Default)

	require.Len(t, wd.Steps, 3)

	step1 := wd.Steps[0]
	require.Equal(t, "step-1", step1.Name)
	require.IsType(t, &ContainerWorkflowStep{}, step1.Variant)

	step2 := wd.Steps[1]
	require.Equal(t, "step-2", step2.Name)
	require.IsType(t, &ContainerWorkflowStep{}, step2.Variant)

	variant := step2.Variant.(*ContainerWorkflowStep)
	require.Equal(t, "relaysh/core:latest", variant.Image)
	require.Len(t, variant.Spec, 5)

	approval1 := wd.Steps[2]
	require.Equal(t, "approval-1", approval1.Name)
	require.IsType(t, &ApprovalWorkflowStep{}, approval1.Variant)
}

func TestYAMLDecoder(t *testing.T) {
	ctx := context.Background()

	fs, err := filepath.Glob("testdata/*.yaml")
	require.NoError(t, err)

	workflows := []string{}

	for _, file := range fs {
		if !strings.HasSuffix(file, "_invalid.yaml") {
			workflows = append(workflows, file)
		}
	}

	// specialCases is a small sample of workflows to inspect to make sure
	// decoding is properly filling out fields. The map key is the filename
	// loaded from ./testdata.
	var specialCases = map[string]func(*testing.T, *WorkflowData){
		"valid.yaml":       validWorkflow,
		"complicated.yaml": complicatedWorkflow,
	}

	yd := YAMLDecoder{}

	for _, file := range workflows {
		basename := filepath.Base(file)

		t.Run(basename, func(t *testing.T) {
			b, err := ioutil.ReadFile(file)
			require.NoError(t, err)

			wd, err := yd.Decode(ctx, b)
			require.NoError(t, err)

			if sc, ok := specialCases[basename]; ok {
				sc(t, wd)
			}
		})
	}
}
