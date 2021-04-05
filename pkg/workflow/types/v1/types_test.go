package v1_test

import (
	"encoding/json"
	"testing"

	v1 "github.com/puppetlabs/relay-core/pkg/workflow/types/v1"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestWorkflowParameterSerialization(t *testing.T) {
	yamlToJSONUnmarshaler := func(data []byte, into interface{}) error {
		var ir v1.WorkflowParameter
		if err := yaml.Unmarshal(data, &ir); err != nil {
			return err
		}

		b, err := json.Marshal(ir)
		if err != nil {
			return err
		}

		return json.Unmarshal(b, into)
	}

	jsonToYAMLUnmarshaler := func(data []byte, into interface{}) error {
		var ir v1.WorkflowParameter
		if err := json.Unmarshal(data, &ir); err != nil {
			return err
		}

		b, err := yaml.Marshal(ir)
		if err != nil {
			return err
		}

		return yaml.Unmarshal(b, into)
	}

	tests := []struct {
		Name      string
		Input     string
		Unmarshal func([]byte, interface{}) error
		Expected  *v1.WorkflowParameter
	}{
		{
			Name:      "JSON with default string",
			Input:     `{"description": "test", "default": "hello"}`,
			Unmarshal: json.Unmarshal,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault("hello"),
		},
		{
			Name:      "YAML with default string",
			Input:     `{description: test, default: hello}`,
			Unmarshal: yaml.Unmarshal,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault("hello"),
		},
		{
			Name:      "round-trip JSON to YAML with default string",
			Input:     `{"description": "test", "default": "hello"}`,
			Unmarshal: jsonToYAMLUnmarshaler,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault("hello"),
		},
		{
			Name:      "round-trip YAML to JSON with default string",
			Input:     `{description: test, default: hello}`,
			Unmarshal: yamlToJSONUnmarshaler,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault("hello"),
		},
		{
			Name:      "JSON with no default",
			Input:     `{"description": "test"}`,
			Unmarshal: json.Unmarshal,
			Expected: &v1.WorkflowParameter{
				Description: "test",
			},
		},
		{
			Name:      "YAML with no default",
			Input:     `{description: test}`,
			Unmarshal: yaml.Unmarshal,
			Expected: &v1.WorkflowParameter{
				Description: "test",
			},
		},
		{
			Name:      "round-trip JSON to YAML with no default",
			Input:     `{"description": "test"}`,
			Unmarshal: jsonToYAMLUnmarshaler,
			Expected: &v1.WorkflowParameter{
				Description: "test",
			},
		},
		{
			Name:      "round-trip YAML to JSON with no default",
			Input:     `{description: test}`,
			Unmarshal: yamlToJSONUnmarshaler,
			Expected: &v1.WorkflowParameter{
				Description: "test",
			},
		},
		{
			Name:      "JSON with default null literal",
			Input:     `{"description": "test", "default": null}`,
			Unmarshal: json.Unmarshal,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault(nil),
		},
		{
			Name:      "YAML with default null literal",
			Input:     `{description: test, default: null}`,
			Unmarshal: yaml.Unmarshal,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault(nil),
		},
		{
			Name:      "round-trip JSON to YAML with default null literal",
			Input:     `{"description": "test", "default": null}`,
			Unmarshal: jsonToYAMLUnmarshaler,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault(nil),
		},
		{
			Name:      "round-trip YAML to JSON with default null literal",
			Input:     `{description: test, "default": null}`,
			Unmarshal: yamlToJSONUnmarshaler,
			Expected: (&v1.WorkflowParameter{
				Description: "test",
			}).WithDefault(nil),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			wp := &v1.WorkflowParameter{}
			require.NoError(t, test.Unmarshal([]byte(test.Input), wp))
			require.Equal(t, test.Expected, wp)
		})
	}
}
