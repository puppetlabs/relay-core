package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/manager/memory"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/stretchr/testify/require"
)

func TestGetSpec(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
		Connections: map[memory.ConnectionKey]map[string]interface{}{
			{Type: "aws", Name: "test-aws-connection"}: {
				"accessKeyID":     "testaccesskey",
				"secretAccessKey": "testsecretaccesskey",
			},
		},
		Secrets: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		Runs: map[string]*opt.SampleConfigRun{
			"test": {
				Parameters: map[string]interface{}{
					"nice-parameter": "nice-value",
				},
				Steps: map[string]*opt.SampleConfigStep{
					"previous-task": {
						Outputs: map[string]interface{}{
							"test-output-key": "test-output-value",
							"test-structured-output-key": map[string]interface{}{
								"a":       "value",
								"another": "thing",
							},
						},
					},
					"current-task": {
						Spec: opt.SampleConfigSpec{
							"superParameter": spec.YAMLTree{
								Tree: "${parameters.nice-parameter}",
							},
							"superSecret": spec.YAMLTree{
								Tree: "${secrets.test-secret-key}",
							},
							"superOutput": spec.YAMLTree{
								Tree: "${outputs.previous-task.test-output-key}",
							},
							"structuredOutput": spec.YAMLTree{
								Tree: "${outputs.previous-task.test-structured-output-key}",
							},
							"superConnection": spec.YAMLTree{
								Tree: "${connections.aws.test-aws-connection}",
							},
							"mergedConnection": spec.YAMLTree{
								Tree: "${merge(connections.aws.test-aws-connection, {'merge': 'me'})}",
							},
							"superNormal": spec.YAMLTree{
								Tree: "test-normal-value",
							},
							"superExpansion": spec.YAMLTree{
								Tree: "${$}",
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

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	// Request the whole spec.
	req, err := http.NewRequest(http.MethodGet, "/spec", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	var r api.GetSpecResponseEnvelope
	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, map[string]interface{}{
		"superParameter": "nice-value",
		"superSecret":    "test-secret-value",
		"superOutput":    "test-output-value",
		"structuredOutput": map[string]interface{}{
			"a":       "value",
			"another": "thing",
		},
		"superConnection": map[string]interface{}{
			"accessKeyID":     "testaccesskey",
			"secretAccessKey": "testsecretaccesskey",
		},
		"mergedConnection": map[string]interface{}{
			"accessKeyID":     "testaccesskey",
			"secretAccessKey": "testsecretaccesskey",
			"merge":           "me",
		},
		"superNormal": "test-normal-value",
		"superExpansion": map[string]interface{}{
			"connections": map[string]interface{}{
				"aws": map[string]interface{}{
					"test-aws-connection": map[string]interface{}{
						"accessKeyID":     "testaccesskey",
						"secretAccessKey": "testsecretaccesskey",
					},
				},
			},
			"outputs": map[string]interface{}{
				"previous-task": map[string]interface{}{
					"test-output-key": "test-output-value",
					"test-structured-output-key": map[string]interface{}{
						"a":       "value",
						"another": "thing",
					},
				},
			},
			"parameters": map[string]interface{}{
				"nice-parameter": "nice-value",
			},
			"secrets": map[string]interface{}{
				"test-secret-key": "test-secret-value",
			},
		},
	}, r.Value.Data)
	require.True(t, r.Complete)

	// Request a specific expression from the spec.
	req.URL.RawQuery = url.Values{"q": []string{"structuredOutput.a"}}.Encode()

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	r = api.GetSpecResponseEnvelope{}
	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, "value", r.Value.Data)
	require.True(t, r.Complete)

	// Request a specific expression from the spec using the JSON path query language
	req.URL.RawQuery = url.Values{
		"q":    []string{"$.structuredOutput.a"},
		"lang": []string{"jsonpath"},
	}.Encode()

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	r = api.GetSpecResponseEnvelope{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&r))
	require.Equal(t, "value", r.Value.Data)
	require.True(t, r.Complete)
}
