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

func TestPostDecorator(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
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

	a := sample.NewAuthenticator(sc, tokenGenerator.Key())
	h := api.NewHandler(a)

	values := map[string]interface{}{
		"type":        string(model.DecoratorTypeLink),
		"description": "a test description",
		"uri":         "https://unit-testing.relay.sh/decorator-location",
	}

	buf := bytes.Buffer{}
	require.NoError(t, json.NewEncoder(&buf).Encode(values))

	req, err := http.NewRequest(http.MethodPost, "/decorators/test-decorator", &buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+testTaskToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusCreated, resp.Result().StatusCode)

	c, err := a.Authenticate(req)
	require.NoError(t, err)

	m := c.Managers
	dm := m.StepDecorators()

	decs, err := dm.List(ctx)
	require.NoError(t, err)
	require.Len(t, decs, 1)

	sd := decs[0]

	require.Equal(t, "test-decorator", sd.Value.Name)
	require.NotNil(t, sd.Value.Link)
	require.Equal(t, "a test description", sd.Value.Link.Description)
	require.Equal(t, "https://unit-testing.relay.sh/decorator-location", sd.Value.Link.URI)
}
