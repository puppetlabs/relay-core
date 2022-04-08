package entrypoint_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/stretchr/testify/require"
)

type mockMetadataAPIOptions struct {
	Delay time.Duration
}

func withMockMetadataAPI(t *testing.T, fn func(ts *httptest.Server), opts mockMetadataAPIOptions) {
	seed := make(map[string]time.Time)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler, _ := shiftPath(r.URL.Path)
		switch handler {
		case "environment", "validate", "logs", "timers":
			if _, ok := seed[handler]; !ok {
				seed[handler] = time.Now()
			}
			if time.Now().After(seed[handler].Add(opts.Delay)) {
				w.WriteHeader(http.StatusOK)
				seed[handler] = time.Now()
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
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
			Config: &entrypoint.Config{
				DeploymentEnvironment: &model.DeploymentEnvironmentTest,
			},
		},
	}

	err := e.Go()
	require.NoError(t, err)
}

func TestEntrypointRunnerWithInvalidMetadataAPIURL(t *testing.T) {
	e := entrypoint.Entrypointer{
		Entrypoint: "ls",
		Args:       []string{"-la"},
		Runner: &entrypoint.RealRunner{
			Config: &entrypoint.Config{
				DeploymentEnvironment: &model.DeploymentEnvironmentTest,
				MetadataAPIURL:        &url.URL{Scheme: "http", Host: "invalid"},
			},
		},
	}

	err := e.Go()
	require.NoError(t, err)
}

func TestEntrypointRunnerWithMockMetadataAPIURL(t *testing.T) {
	opts := mockMetadataAPIOptions{
		Delay: 250 * time.Millisecond,
	}

	withMockMetadataAPI(t, func(ts *httptest.Server) {
		u, err := url.Parse(ts.URL)
		require.NoError(t, err)

		e := entrypoint.Entrypointer{
			Entrypoint: "ls",
			Args:       []string{"-la"},
			Runner: &entrypoint.RealRunner{
				Config: &entrypoint.Config{
					DeploymentEnvironment: &model.DeploymentEnvironmentTest,
					MetadataAPIURL:        u,
				},
			},
		}

		err = e.Go()
		require.NoError(t, err)
	}, opts)
}
