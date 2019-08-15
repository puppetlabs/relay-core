package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiutil "github.com/puppetlabs/horsehead/httputil/api"
)

type MockMetadataAPIOptions struct {
	Name           string
	ResponseObject interface{}
}

func WithMockMetadataAPI(t *testing.T, fn func(ts *httptest.Server), opts MockMetadataAPIOptions) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, fmt.Sprintf("/specs/%s", opts.Name)) {
			apiutil.WriteObjectOK(r.Context(), w, opts.ResponseObject)

			return
		}

		http.NotFound(w, r)
		return
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	fn(ts)
}
