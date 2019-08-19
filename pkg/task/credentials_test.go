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

func TestCredentialOutput(t *testing.T) {
	t.Skip("Functional testing harness. Needs to be completed.")

	credentialSpec := make(map[string]string)

	data := base64.StdEncoding.EncodeToString([]byte("testdata"))
	credentialSpec["ca.pem"] = data
	credentialSpec["key.pem"] = data

	testSpec := &model.CredentialSpec{
		Credentials: credentialSpec,
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

		err := task.ProcessCredentials("output")
		require.Nil(t, err, "err is not nil")
	}, opts)
}
