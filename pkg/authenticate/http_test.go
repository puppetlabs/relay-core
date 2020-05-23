package authenticate_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/stretchr/testify/require"
)

func TestHTTPAuthorizationHeaderIntermediary(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		AuthorizationHeader string
		ExpectedRaw         authenticate.Raw
		ExpectedError       error
	}{
		{
			AuthorizationHeader: "Bearer test-token",
			ExpectedRaw:         authenticate.Raw("test-token"),
		},
		{
			AuthorizationHeader: "Basic Zm9vOmJhcg==",
			ExpectedError:       &authenticate.NotFoundError{Reason: "http: username not empty"},
		},
		{
			AuthorizationHeader: "Basic Om15LXRva2Vu",
			ExpectedRaw:         authenticate.Raw("my-token"),
		},
		{
			ExpectedError: &authenticate.NotFoundError{Reason: "http: neither Basic nor Bearer authentication present"},
		},
	}
	for _, test := range tests {
		t.Run(test.AuthorizationHeader, func(t *testing.T) {
			h := make(http.Header)
			if test.AuthorizationHeader != "" {
				h.Set("Authorization", test.AuthorizationHeader)
			}

			raw, err := authenticate.NewHTTPAuthorizationHeaderIntermediary(&http.Request{
				Header: h,
			}).Next(ctx, authenticate.NewAuthentication())
			require.Equal(t, test.ExpectedRaw, raw)
			require.Equal(t, test.ExpectedError, err)
		})
	}
}
