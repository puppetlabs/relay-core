package api_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestPostLog(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
		Secrets: map[string]string{
			"test-secret-key": "test-secret-value",
		},
		Runs: map[string]*opt.SampleConfigRun{
			"test": {
				Steps: map[string]*opt.SampleConfigStep{
					"previous-task": {
						Outputs: map[string]interface{}{
							"test-output-key": "test-output-value",
						},
					},
					"current-task": {
						Spec: opt.SampleConfigSpec{
							"superSecret": spec.YAMLTree{
								Tree: "${secrets.test-secret-key}",
							},
							"superOutput": spec.YAMLTree{
								Tree: "${outputs.previous-task.test-output-key}",
							},
						},
						Env: opt.SampleConfigEnvironment{
							"test-environment-variable-from-secret": spec.YAMLTree{
								Tree: "${secrets.test-secret-key}",
							},
							"test-environment-variable-from-output": spec.YAMLTree{
								Tree: "${outputs.previous-task.test-output-key}",
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

	ls := &plspb.LogCreateRequest{
		Context: "current-task",
		Name:    "stdout",
	}

	buf, err := proto.Marshal(ls)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/logs", bytes.NewBuffer(buf))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

	log := &plspb.LogCreateResponse{}
	err = proto.Unmarshal(resp.Body.Bytes(), log)
	require.NoError(t, err)

	// FIXME Create a new log ID for now (rather than use the response)
	// Until a reliable means of mocking the service is available
	// This testing is primarily for the means of validating the API (not the service)
	id := uuid.New().String()

	for i := 0; i < 10; i++ {
		u := &url.URL{Path: fmt.Sprintf("/logs/%s/messages", id)}

		lm := &plspb.LogMessageAppendRequest{
			LogId:   id,
			Payload: []byte(uuid.New().String()),
		}

		buf, err := proto.Marshal(lm)
		require.NoError(t, err)

		req, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(buf))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+currentTaskToken)
		req.Header.Set("Content-Type", "application/octet-stream")

		resp = httptest.NewRecorder()
		h.ServeHTTP(resp, req)
		require.Equal(t, http.StatusAccepted, resp.Result().StatusCode)

		out := &plspb.LogMessageAppendResponse{}
		err = proto.Unmarshal(resp.Body.Bytes(), out)
		require.NoError(t, err)
	}
}
