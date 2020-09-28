package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/errors"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetState(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	tests := []struct {
		Name          string
		State         map[string]interface{}
		ExpectedValue interface{}
		ExpectedError errawr.Error
	}{
		{
			Name: "Basic",
			State: map[string]interface{}{
				"test-key": "test-value",
			},
			ExpectedValue: "test-value",
		},
		{
			Name: "Non-UTF-8 characters",
			State: map[string]interface{}{
				"test-key": "hello\x90",
			},
			ExpectedValue: "hello\x90",
		},
		{
			Name: "Nonexistent",
			State: map[string]interface{}{
				"other-key": "test-value",
			},
			ExpectedError: errors.NewModelNotFoundError(),
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			sc := &opt.SampleConfig{
				Runs: map[string]*opt.SampleConfigRun{
					"test": &opt.SampleConfigRun{
						Steps: map[string]*opt.SampleConfigStep{
							"test-task": &opt.SampleConfigStep{
								State: test.State,
							},
						},
					},
				},
			}

			tokenMap := tokenGenerator.GenerateAll(ctx, sc)

			testTaskToken, found := tokenMap.ForStep("test", "test-task")
			require.True(t, found)

			h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()), nil)

			req, err := http.NewRequest(http.MethodGet, "/state/test-key", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+testTaskToken)

			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			if test.ExpectedError == nil {
				require.Equal(t, http.StatusOK, resp.Result().StatusCode)

				var env api.GetStateResponseEnvelope
				require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&env))
				require.Equal(t, test.ExpectedValue, env.Value.Data)
			} else {
				testutil.RequireErrorResponse(t, test.ExpectedError, resp.Result())
			}
		})
	}
}
