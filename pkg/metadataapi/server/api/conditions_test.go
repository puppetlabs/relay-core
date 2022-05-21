package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetConditions(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	tests := []struct {
		Name             string
		Conditions       any
		ExpectedError    errawr.Error
		ExpectedResolved bool
		ExpectedSuccess  bool
	}{
		{
			Name: "Success",
			Conditions: []interface{}{
				"${outputs.previous-task.output1 == 'foobar'}",
				"${outputs.previous-task.output1 != 'barfoo'}",
			},
			ExpectedResolved: true,
			ExpectedSuccess:  true,
		},
		{
			Name: "Failure",
			Conditions: []interface{}{
				"${outputs.previous-task.output1 == 'foobar'}",
				"${outputs.previous-task.output1 != 'foobar'}",
			},
			ExpectedResolved: true,
			ExpectedSuccess:  false,
		},
		{
			Name:             "Explicitly true",
			Conditions:       true,
			ExpectedResolved: true,
			ExpectedSuccess:  true,
		},
		{
			Name:             "Explicitly false",
			Conditions:       false,
			ExpectedResolved: true,
			ExpectedSuccess:  false,
		},
		{
			Name:             "Resolution error (single expression)",
			Conditions:       "${parameters.param1 == 'foobar'}",
			ExpectedResolved: false,
			ExpectedSuccess:  false,
		},
		{
			Name: "Resolution error (multiple expressions)",
			Conditions: []interface{}{
				"${parameters.param1 == 'foobar'}",
			},
			ExpectedResolved: false,
			ExpectedSuccess:  false,
		},
		{
			Name:       "Condition type error (single string)",
			Conditions: "foobar",
			ExpectedError: errors.NewConditionTypeError(
				`string`,
			),
		},
		{
			Name: "Condition type error (multiple strings)",
			Conditions: []interface{}{
				"foobar",
				"fubar",
			},
			ExpectedError: errors.NewConditionTypeError(
				`string`,
			),
		},
		{
			Name: "Short-circuit failure ordering variant 1 (failure first)",
			Conditions: []interface{}{
				"${outputs.previous-task.output1 == 'fubar'}",
				"${parameters.param1 == 'fubar'}",
			},
			ExpectedResolved: true,
			ExpectedSuccess:  false,
		},
		{
			Name: "Short-circuit failure ordering variant 2 (unresolvable first)",
			Conditions: []interface{}{
				"${parameters.param1 == 'fubar'}",
				"${outputs.previous-task.output1 == 'fubar'}",
			},
			ExpectedResolved: true,
			ExpectedSuccess:  false,
		},
		{
			Name:             "No conditions are defined",
			Conditions:       nil,
			ExpectedResolved: true,
			ExpectedSuccess:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			sc := &opt.SampleConfig{
				Runs: map[string]*opt.SampleConfigRun{
					"test": {
						Steps: map[string]*opt.SampleConfigStep{
							"previous-task": {},
							"current-task": {
								Conditions: spec.YAMLTree{
									Tree: test.Conditions,
								},
							},
						},
					},
				},
			}

			tokenMap := tokenGenerator.GenerateAll(ctx, sc)

			previousTaskToken, found := tokenMap.ForStep("test", "previous-task")
			require.True(t, found)

			currentTaskToken, found := tokenMap.ForStep("test", "current-task")
			require.True(t, found)

			h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

			req, err := http.NewRequest(http.MethodPut, "/outputs/output1", strings.NewReader("foobar"))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+previousTaskToken)

			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

			req, err = http.NewRequest(http.MethodGet, "/conditions", http.NoBody)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+currentTaskToken)

			resp = httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			if test.ExpectedError == nil {
				require.Equal(t, http.StatusOK, resp.Result().StatusCode)

				var env api.GetConditionsResponseEnvelope
				require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&env))
				require.Equal(t, test.ExpectedResolved, env.Resolved)
				require.Equal(t, test.ExpectedSuccess, env.Success)
			} else {
				testutil.RequireErrorResponse(t, test.ExpectedError, resp.Result())
			}
		})
	}
}
