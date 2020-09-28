package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/stretchr/testify/require"
)

func TestPutGetOutput(t *testing.T) {
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

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()), nil)

	// Set an output.
	req, err := http.NewRequest(http.MethodPut, "/outputs/foo", strings.NewReader("bar\x90"))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

	// Read it back.
	req, err = http.NewRequest(http.MethodGet, "/outputs/test-task/foo", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testTaskToken)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Result().StatusCode)

	var out api.GetOutputResponseEnvelope
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	require.Equal(t, "foo", out.Key)
	require.Equal(t, "test-task", out.TaskName)
	require.Equal(t, "bar\x90", out.Value.Data)
}
