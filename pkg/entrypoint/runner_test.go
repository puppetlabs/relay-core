package entrypoint_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/stretchr/testify/require"
)

type mockMetadataAPIOptions struct {
}

func withMockMetadataAPI(t *testing.T, fn func(ts *httptest.Server), opts mockMetadataAPIOptions) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, _ := shiftPath(r.URL.Path)
		switch handler {
		case "environment":
			w.WriteHeader(http.StatusOK)
			return
		case "validate":
			w.WriteHeader(http.StatusOK)
			return
		case "logs":
			w.WriteHeader(http.StatusOK)
			return
		}

		http.NotFound(w, r)
	})

	ts := httptest.NewServer(handler)
	defer ts.Close()

	fn(ts)
}

// shiftPath takes a URI path and returns the first segment as the head
// and the remaining segments as the tail.
func shiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], ""
	}

	return p[1:i], p[i:]
}

func TestEntrypointRunnerWithoutMetadataAPIURL(t *testing.T) {
	e := entrypoint.Entrypointer{
		Entrypoint: "ls",
		Args:       []string{"-la"},
		Runner: &entrypoint.RealRunner{
			TimeoutLong:  10 * time.Second,
			TimeoutShort: 2 * time.Second,
		},
	}

	err := e.Go()
	require.NoError(t, err)
}

func TestEntrypointRunnerWithInvalidMetadataAPIURL(t *testing.T) {
	os.Setenv(entrypoint.MetadataAPIURLEnvName, "http://hi")

	e := entrypoint.Entrypointer{
		Entrypoint: "ls",
		Args:       []string{"-la"},
		Runner: &entrypoint.RealRunner{
			TimeoutLong:  10 * time.Second,
			TimeoutShort: 2 * time.Second,
		},
	}

	err := e.Go()
	require.NoError(t, err)
}

func TestEntrypointRunnerWithMockMetadataAPIURL(t *testing.T) {
	opts := mockMetadataAPIOptions{}

	withMockMetadataAPI(t, func(ts *httptest.Server) {
		os.Setenv(entrypoint.MetadataAPIURLEnvName, ts.URL)

		e := entrypoint.Entrypointer{
			Entrypoint: "ls",
			Args:       []string{"-la"},
			Runner: &entrypoint.RealRunner{
				TimeoutLong:  10 * time.Second,
				TimeoutShort: 2 * time.Second,
			},
		}

		err := e.Go()
		require.NoError(t, err)
	}, opts)
}
