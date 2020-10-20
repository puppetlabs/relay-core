package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/workflow/validation"
	"github.com/stretchr/testify/require"
)

func WithStepMetadataServer(t *testing.T, metadataPath string, fn func(ts *httptest.Server)) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(metadataPath)
		require.NoError(t, err)

		fi, err := f.Stat()
		require.NoError(t, err)

		w.Header().Set("content-type", "application/json")

		http.ServeContent(w, r, "", fi.ModTime(), f)
	}))

	defer ts.Close()

	fn(ts)
}

func WithStepMetadataSchemaRegistry(t *testing.T, metadataPath string, fn func(reg validation.SchemaRegistry)) {
	WithStepMetadataServer(t, metadataPath, func(ts *httptest.Server) {
		u, err := url.Parse(ts.URL)
		require.NoError(t, err)

		reg, err := validation.NewStepMetadataSchemaRegistry(u)
		require.NoError(t, err)

		fn(reg)
	})
}
