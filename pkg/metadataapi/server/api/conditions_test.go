package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/parse"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/serialize"
	sdktestutil "github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/testutil"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/opt"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/sample"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/api"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetConditions(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	tests := []struct {
		Name            string
		Conditions      parse.Tree
		ExpectedError   errawr.Error
		ExpectedSuccess bool
	}{
		{
			Name: "Success",
			Conditions: []interface{}{
				sdktestutil.JSONInvocation("equals", []interface{}{
					sdktestutil.JSONOutput("previous-task", "output1"),
					"foobar",
				}),
				sdktestutil.JSONInvocation("notEquals", []interface{}{
					sdktestutil.JSONOutput("previous-task", "output1"),
					"barfoo",
				}),
			},
			ExpectedSuccess: true,
		},
		{
			Name: "Failure",
			Conditions: []interface{}{
				sdktestutil.JSONInvocation("equals", []interface{}{
					sdktestutil.JSONOutput("previous-task", "output1"),
					"foobar",
				}),
				sdktestutil.JSONInvocation("notEquals", []interface{}{
					sdktestutil.JSONOutput("previous-task", "output1"),
					"foobar",
				}),
			},
			ExpectedSuccess: false,
		},
		{
			Name: "Resolution error",
			Conditions: []interface{}{
				sdktestutil.JSONInvocation("equals", []interface{}{
					sdktestutil.JSONParameter("param1"),
					"foobar",
				}),
			},
			ExpectedError: errors.NewExpressionUnresolvableError([]string{
				`resolve: parameter "param1" could not be found`,
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			sc := &opt.SampleConfig{
				Runs: map[string]*opt.SampleConfigRun{
					"test": &opt.SampleConfigRun{
						Steps: map[string]*opt.SampleConfigStep{
							"previous-task": &opt.SampleConfigStep{},
							"current-task": &opt.SampleConfigStep{
								Conditions: serialize.YAMLTree{
									Tree: test.Conditions,
								},
							},
						},
					},
				},
			}

			tokenMap := tokenGenerator.GenerateAll(ctx, sc)

			previousTaskToken, found := tokenMap.Get("test", "previous-task")
			require.True(t, found)

			currentTaskToken, found := tokenMap.Get("test", "current-task")
			require.True(t, found)

			h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

			// Set the output so the condition can succeed.
			req, err := http.NewRequest(http.MethodPut, "/outputs/output1", strings.NewReader("foobar"))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+previousTaskToken)

			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

			// Query the condition to find out whether it succeeded.
			req, err = http.NewRequest(http.MethodGet, "/conditions", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+currentTaskToken)

			resp = httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			if test.ExpectedError == nil {
				require.Equal(t, http.StatusOK, resp.Result().StatusCode)

				var env api.GetConditionsResponseEnvelope
				require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&env))
				require.Equal(t, test.ExpectedSuccess, env.Success)
			} else {
				testutil.RequireErrorResponse(t, test.ExpectedError, resp.Result())
			}
		})
	}
}
