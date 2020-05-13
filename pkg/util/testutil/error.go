package testutil

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/horsehead/v2/httputil/api"
	"github.com/stretchr/testify/assert"
)

func AssertErrorResponse(t *testing.T, expected errawr.Error, resp *http.Response) bool {
	expectedStatusCode := http.StatusInternalServerError
	if hm, ok := expected.Metadata().HTTP(); ok {
		expectedStatusCode = hm.Status()
	}

	if !assert.Equal(t, expectedStatusCode, resp.StatusCode) {
		return false
	}

	var env api.ErrorEnvelope
	if !assert.NoError(t, json.NewDecoder(resp.Body).Decode(&env)) {
		return false
	}

	if !assert.Equal(t, expected.Error(), env.Error.AsError().Error()) {
		return false
	}

	return true
}

func RequireErrorResponse(t *testing.T, expected errawr.Error, resp *http.Response) {
	if !AssertErrorResponse(t, expected, resp) {
		t.FailNow()
	}
}
