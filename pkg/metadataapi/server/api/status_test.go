package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestPutActionStatus(t *testing.T) {
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

	as := &model.ActionStatus{
		ExitCode: 1,
	}

	buf := new(bytes.Buffer)
	err = json.NewEncoder(buf).Encode(as)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPut, "/status", buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)
}
