package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/opt"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/sample"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/api"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
)

func TestGetSecret(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
		Secrets: map[string]string{
			"foo": "bar\x90",
			"baz": "",
		},
		Runs: map[string]*opt.SampleConfigRun{
			"test": &opt.SampleConfigRun{
				Steps: map[string]*opt.SampleConfigStep{
					"test-task": &opt.SampleConfigStep{},
				},
			},
		},
	}

	tokenMap := tokenGenerator.GenerateAll(ctx, sc)

	testTaskToken, found := tokenMap.ForStep("test", "test-task")
	require.True(t, found)

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	tests := []struct {
		SecretName    string
		ExpectedValue string
		ExpectedError errawr.Error
	}{
		{
			SecretName:    "foo",
			ExpectedValue: "bar\x90",
		},
		{
			SecretName:    "baz",
			ExpectedValue: "",
		},
		{
			SecretName:    "quux",
			ExpectedError: errors.NewModelNotFoundError(),
		},
	}
	for _, test := range tests {
		t.Run(test.SecretName, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("/secrets/%s", test.SecretName), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+testTaskToken)

			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)
			if test.ExpectedError == nil {
				require.Equal(t, http.StatusOK, resp.Result().StatusCode)

				var env api.GetSecretResponseEnvelope
				require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&env))

				b, err := env.Value.Decode()
				require.NoError(t, err)
				require.Equal(t, []byte(test.ExpectedValue), b)
			} else {
				testutil.RequireErrorResponse(t, test.ExpectedError, resp.Result())
			}
		})
	}
}
