package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/model"
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
			"test": {
				Steps: map[string]*opt.SampleConfigStep{
					"test-task": {},
				},
			},
		},
	}

	tokenMap := tokenGenerator.GenerateAll(ctx, sc)

	testTaskToken, found := tokenMap.ForStep("test", "test-task")
	require.True(t, found)

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	metadata := &model.StepOutputMetadata{
		Sensitive: true,
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(metadata)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "/outputs/foo/metadata", buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

	req, err = http.NewRequest(http.MethodPut, "/outputs/foo", strings.NewReader("bar\x90"))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testTaskToken)

	resp = httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

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
	require.Equal(t, true, out.Metadata.Sensitive)
}
