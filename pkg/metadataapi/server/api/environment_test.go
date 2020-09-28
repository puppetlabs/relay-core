package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	sdktestutil "github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/stretchr/testify/require"
)

func TestGetEnvironment(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
		Secrets: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		Runs: map[string]*opt.SampleConfigRun{
			"test": &opt.SampleConfigRun{
				Steps: map[string]*opt.SampleConfigStep{
					"previous-task": &opt.SampleConfigStep{
						Outputs: map[string]interface{}{
							"test-output-key": "test-output-value",
						},
					},
					"current-task": &opt.SampleConfigStep{
						Spec: opt.SampleConfigSpec{
							"superSecret": serialize.YAMLTree{
								Tree: sdktestutil.JSONSecret("test-secret-key"),
							},
							"superOutput": serialize.YAMLTree{
								Tree: sdktestutil.JSONOutput("previous-task", "test-output-key"),
							},
						},
						Env: opt.SampleConfigEnvironment{
							"test-environment-variable-from-secret": serialize.YAMLTree{
								Tree: sdktestutil.JSONSecret("test-secret-key"),
							},
							"test-environment-variable-from-output": serialize.YAMLTree{
								Tree: sdktestutil.JSONOutput("previous-task", "test-output-key"),
							},
						},
					},
				},
			},
		},
	}

	tokenMap := tokenGenerator.GenerateAll(ctx, sc)

	currentTaskToken, found := tokenMap.ForStep("test", "current-task")
	require.True(t, found)

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()), nil)

	req, err := http.NewRequest(http.MethodGet, "/spec", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	var r evaluate.JSONResultEnvelope
	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, map[string]interface{}{
		"superSecret": "test-secret-value",
		"superOutput": "test-output-value",
	}, r.Value.Data)
	require.True(t, r.Complete)

	req, err = http.NewRequest(http.MethodGet, "/environment", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, map[string]interface{}{
		"test-environment-variable-from-secret": "test-secret-value",
		"test-environment-variable-from-output": "test-output-value",
	}, r.Value.Data)
	require.True(t, r.Complete)

	req, err = http.NewRequest(http.MethodGet, "/environment/test-environment-variable-from-secret", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, "test-secret-value", r.Value.Data)

	req, err = http.NewRequest(http.MethodGet, "/environment/test-environment-variable-from-output", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	require.NoError(t, json.NewDecoder(resp.Body).Decode(&r))
	require.Equal(t, "test-output-value", r.Value.Data)
}
