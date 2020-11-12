package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/expr/serialize"
	sdktestutil "github.com/puppetlabs/relay-core/pkg/expr/testutil"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/puppetlabs/relay-pls/pkg/plspb"
	"github.com/stretchr/testify/require"
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

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	ls := &plspb.LogCreateRequest{
		Context: "current-task",
		Name:    "stdout",
	}

	buf, err := json.Marshal(ls)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "/logs", bytes.NewBuffer(buf))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+currentTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

	log := &plspb.LogCreateResponse{}
	err = json.NewDecoder(resp.Body).Decode(log)
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		u := &url.URL{Path: fmt.Sprintf("/logs/messages")}

		lm := &plspb.LogMessageAppendRequest{
			LogId:   log.GetLogId(),
			Payload: []byte(uuid.New().String()),
		}

		buf, err := json.Marshal(lm)
		require.NoError(t, err)

		req, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBuffer(buf))
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+currentTaskToken)

		resp = httptest.NewRecorder()
		h.ServeHTTP(resp, req)
		require.Equal(t, http.StatusAccepted, resp.Result().StatusCode)

		out := &plspb.LogMessageAppendResponse{}
		err = json.NewDecoder(resp.Body).Decode(out)
		require.NoError(t, err)

		require.NotEmpty(t, out.GetLogMessageId())
		require.Equal(t, log.GetLogId(), out.GetLogId())
	}
}
