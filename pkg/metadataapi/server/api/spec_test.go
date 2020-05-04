package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/evaluate"
	"github.com/puppetlabs/nebula-sdk/pkg/workflow/spec/serialize"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/opt"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/sample"
	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/api"
	"github.com/stretchr/testify/require"
)

func TestGetSpec(t *testing.T) {
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
							"test-structured-output-key": map[string]interface{}{
								"a":       "value",
								"another": "thing",
							},
						},
					},
					"current-task": &opt.SampleConfigStep{
						Spec: opt.SampleConfigSpec{
							"superSecret": serialize.YAMLTree{
								Tree: map[string]interface{}{
									"$type": "Secret",
									"name":  "test-secret-key",
								},
							},
							"superOutput": serialize.YAMLTree{
								Tree: map[string]interface{}{
									"$type": "Output",
									"from":  "previous-task",
									"name":  "test-output-key",
								},
							},
							"structuredOutput": serialize.YAMLTree{
								Tree: map[string]interface{}{
									"$type": "Output",
									"from":  "previous-task",
									"name":  "test-structured-output-key",
								},
							},
							"superNormal": serialize.YAMLTree{
								Tree: "test-normal-value",
							},
						},
					},
				},
			},
		},
	}

	tokenMap := tokenGenerator.GenerateAll(ctx, sc)

	currentTaskToken, found := tokenMap.Get("test", "current-task")
	require.True(t, found)

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	// Request the whole spec.
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
		"structuredOutput": map[string]interface{}{
			"a":       "value",
			"another": "thing",
		},
		"superNormal": "test-normal-value",
	}, r.Value.Data)
	require.True(t, r.Complete)

	// Request a specific expression from the spec.
	req.URL.RawQuery = url.Values{"q": []string{"structuredOutput.a"}}.Encode()

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	r = evaluate.JSONResultEnvelope{}
	require.NoError(t, json.NewDecoder(resp.Result().Body).Decode(&r))
	require.Equal(t, "value", r.Value.Data)
	require.True(t, r.Complete)
}
