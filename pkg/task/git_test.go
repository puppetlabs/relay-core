package task

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/puppetlabs/nebula-tasks/pkg/model"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/puppetlabs/nebula-tasks/pkg/testutil"
)

func TestGitOutput(t *testing.T) {
	t.Skip("Functional testing harness. Needs to be completed.")

	testSpec := &model.GitSpec{
		GitRepository: &model.GitDetails{
			SSHKey:     "<ssh_key>",
			KnownHosts: "<known_hosts>",
			Name:       "<name>",
			Repository: "<repository>",
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

		err := task.CloneRepository("master", "output")
		require.Nil(t, err, "err is not nil")
	}, opts)
}
