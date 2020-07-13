package v1_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/util/typeutil"
	v1 "github.com/puppetlabs/relay-core/pkg/workflow/types/v1"
	"github.com/stretchr/testify/require"
)

func TestFixtureValidation(t *testing.T) {
	fs, err := filepath.Glob("testdata/*.yaml")
	require.NoError(t, err)

	for _, file := range fs {
		t.Run(filepath.Base(file), func(t *testing.T) {
			b, err := ioutil.ReadFile(file)
			require.NoError(t, err)

			err = v1.ValidateYAML(string(b))
			if strings.HasSuffix(file[:len(file)-len(filepath.Ext(file))], "_invalid") {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFixtureValidationForTriggers(t *testing.T) {
	tcs := []struct {
		Name          string
		File          string
		ExpectedError error
	}{
		{
			Name:          "Valid Triggers",
			File:          "testdata/triggers/triggers.yaml",
			ExpectedError: nil,
		},
		{
			Name: "Invalid Triggers: Binding",
			File: "testdata/triggers/triggers_with_invalid_binding.yaml",
			ExpectedError: &typeutil.ValidationError{
				FieldErrors: []*typeutil.FieldValidationError{
					{
						Context:     "(root).triggers.0.binding",
						Field:       "triggers.0.binding",
						Description: "Invalid type. Expected: object, given: array",
						Type:        "invalid_type",
					},
				},
			},
		},
		{
			Name: "Invalid Triggers: Source",
			File: "testdata/triggers/triggers_with_invalid_source.yaml",
			ExpectedError: &typeutil.ValidationError{
				FieldErrors: []*typeutil.FieldValidationError{
					{
						Context:     "(root).triggers.0.source",
						Field:       "triggers.0.source",
						Description: "schedule is required",
						Type:        "required",
					},
					{
						Context:     "(root).triggers.1.source",
						Field:       "triggers.1.source",
						Description: "image is required",
						Type:        "required",
					},
				},
			},
		},
		{
			Name: "Invalid Triggers: Source Type",
			File: "testdata/triggers/triggers_with_invalid_source_type.yaml",
			ExpectedError: &typeutil.ValidationError{
				FieldErrors: []*typeutil.FieldValidationError{
					{
						Context:     "(root).triggers.0.source.type",
						Field:       "triggers.0.source.type",
						Description: "triggers.0.source.type does not match: \"schedule\"",
						Type:        "const",
					},
					{
						Context:     "(root).triggers.1.source.type",
						Field:       "triggers.1.source.type",
						Description: "triggers.1.source.type does not match: \"push\"",
						Type:        "const",
					},
					{
						Context:     "(root).triggers.2.source.type",
						Field:       "triggers.2.source.type",
						Description: "triggers.2.source.type does not match: \"webhook\"",
						Type:        "const",
					},
				},
			},
		},
	}
	for _, test := range tcs {
		t.Run(fmt.Sprintf("%s", test.Name), func(t *testing.T) {
			b, err := ioutil.ReadFile(test.File)
			require.NoError(t, err)

			err = v1.ValidateYAML(string(b))
			if test.ExpectedError != nil {
				require.Error(t, err)
				if test.ExpectedError != nil {
					require.Equal(t, test.ExpectedError, err)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}
