package task

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/puppetlabs/nebula-tasks/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type TestValues struct {
	Name    string   `yaml:"name" json:"name"`
	SomeNum int      `yaml:"someNum" json:"someNum"`
	Data    []string `yaml:"data" json:"data"`
}

type TestSpec struct {
	Values *TestValues `yaml:"values" json:"values"`
}

func TestGetFileOutput(t *testing.T) {
	t.Skip("Functional testing harness. Needs to be completed.")

	testSpec := &TestSpec{
		Values: &TestValues{
			Name:    "test1",
			SomeNum: 12,
			Data:    []string{"something", "else"},
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

		err := task.WriteFile("output/values-test.yml", "values", "yaml")
		require.Nil(t, err, "err is not nil")
	}, opts)
}
