package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/metadataapi/opt"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/sample"
	"github.com/puppetlabs/relay-core/pkg/metadataapi/server/api"
	"github.com/stretchr/testify/require"
)

func TestPostEvent(t *testing.T) {
	ctx := context.Background()

	tokenGenerator, err := sample.NewHS256TokenGenerator(nil)
	require.NoError(t, err)

	sc := &opt.SampleConfig{
		Triggers: map[string]*opt.SampleConfigTrigger{
			"test": {},
		},
	}

	tokenMap := tokenGenerator.GenerateAll(ctx, sc)

	triggerToken, found := tokenMap.ForTrigger("test")
	require.True(t, found)

	h := api.NewHandler(sample.NewAuthenticator(sc, tokenGenerator.Key()))

	req, err := http.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"data":{"foo":"bar"}}`))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+triggerToken)

	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	require.Equal(t, http.StatusAccepted, resp.Result().StatusCode)
}
