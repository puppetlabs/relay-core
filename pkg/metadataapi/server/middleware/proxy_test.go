package middleware_test

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/metadataapi/server/middleware"
	"github.com/stretchr/testify/require"
)

func TestTrustedProxyHops(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.RemoteAddr)
	})

	tests := []struct {
		Name               string
		RemoteAddr         string
		XForwardedFor      string
		TrustedHops        int
		ExpectedRemoteAddr string
	}{
		{
			Name:               "Untrusted",
			RemoteAddr:         "10.20.30.40",
			XForwardedFor:      "192.168.0.1, 192.168.0.2",
			TrustedHops:        0,
			ExpectedRemoteAddr: "10.20.30.40",
		},
		{
			Name:               "One hop trusted",
			RemoteAddr:         "10.20.30.40",
			XForwardedFor:      "192.168.0.1, 192.168.0.2",
			TrustedHops:        1,
			ExpectedRemoteAddr: "192.168.0.2",
		},
		{
			Name:               "Two hops trusted",
			RemoteAddr:         "10.20.30.40",
			XForwardedFor:      "192.168.0.1, 192.168.0.2",
			TrustedHops:        2,
			ExpectedRemoteAddr: "192.168.0.1",
		},
		{
			Name:               "Three hops trusted with less than three hops",
			RemoteAddr:         "10.20.30.40",
			XForwardedFor:      "192.168.0.1, 192.168.0.2",
			TrustedHops:        3,
			ExpectedRemoteAddr: "192.168.0.1",
		},
		{
			Name:               "Invalid address",
			RemoteAddr:         "10.20.30.40",
			XForwardedFor:      "192.168.0.1, foo",
			TrustedHops:        3,
			ExpectedRemoteAddr: "10.20.30.40",
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: test.RemoteAddr,
				Header:     http.Header{},
			}
			req.Header.Set("X-Forwarded-For", test.XForwardedFor)

			mh := middleware.WithTrustedProxyHops(test.TrustedHops)(h)

			resp := httptest.NewRecorder()
			mh.ServeHTTP(resp, req)
			require.Equal(t, http.StatusOK, resp.Result().StatusCode)

			b, err := ioutil.ReadAll(resp.Result().Body)
			require.NoError(t, err)
			require.Equal(t, test.ExpectedRemoteAddr, string(b))
		})
	}
}
