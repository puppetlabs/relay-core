package task

import (
	"encoding/base64"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/puppetlabs/nebula-tasks/pkg/model"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/puppetlabs/nebula-tasks/pkg/testutil"
)

func TestClusterOutput(t *testing.T) {
	t.Skip("Functional testing harness. Needs to be completed.")

	data := base64.StdEncoding.EncodeToString([]byte("cadata"))

	testSpec := &model.ClusterSpec{
		Cluster: &model.ClusterDetails{
			Name:   "test1",
			Token:  "tokendata",
			CAData: data,
			URL:    "url",
		},
	}

	opts := testutil.MockMetadataAPIOptions{
		Name:           "test1",
		ResponseObject: testSpec,
	}

	testutil.WithMockMetadataAPI(t, func(ts *httptest.Server) {
		opts := taskutil.DefaultPlanOptions{
			Client:  ts.Client(),
			SpecURL: fmt.Sprintf("%s/specs/test1", ts.URL),
		}

		task := NewTaskInterface(opts)

		err := task.ProcessClusters("output")
		require.Nil(t, err, "err is not nil")
	}, opts)
}
