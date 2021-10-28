package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	utilapi "github.com/puppetlabs/leg/httputil/api"
	"github.com/puppetlabs/relay-client-go/client/pkg/client/openapi"
	"github.com/puppetlabs/relay-core/pkg/manager/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowRunManager(t *testing.T) {
	ctx := context.Background()

	workflowName := "test-workflow"
	token := "some-token"
	params := map[string]openapi.WorkflowRunParameter{
		"param-1": openapi.WorkflowRunParameter{
			Value: "value-1",
		},
		"param-2": openapi.WorkflowRunParameter{
			Value: "value-2",
		},
	}

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, fmt.Sprintf("/api/workflows/%s/runs", workflowName), r.URL.Path)
		assert.Equal(t, fmt.Sprintf("Bearer %s", token), r.Header.Get("authorization"))

		env := openapi.CreateWorkflowRun{}

		require.NoError(t, json.NewDecoder(r.Body).Decode(&env))
		defer r.Body.Close()

		envParams := *env.Parameters

		require.Len(t, envParams, len(params))

		for k, v := range params {
			got, ok := envParams[k]
			require.True(t, ok)

			require.Equal(t, v.Value, got.Value)
		}

		appURL := r.URL.ResolveReference(&url.URL{Path: fmt.Sprintf("/workflows/%s/runs/2", workflowName)}).String()

		resp := openapi.WorkflowRunEntity{
			Run: &openapi.WorkflowRun{
				RunNumber: 2,
				AppUrl:    &appURL,
			},
			Access: &openapi.EntityAccess{},
		}

		utilapi.WriteObjectCreated(ctx, w, resp)
	}))
	defer s.Close()

	m, err := api.NewWorkflowRunManager(s.URL, token)
	require.NoError(t, err)

	wr, err := m.Run(ctx, workflowName, params)
	require.NoError(t, err)

	require.Equal(t, workflowName, wr.Name)
	require.Equal(t, int32(2), wr.RunNumber)
	require.Equal(t, fmt.Sprintf("/workflows/%s/runs/2", workflowName), wr.URL.Path)
}
