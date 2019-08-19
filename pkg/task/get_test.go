package task

import (
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/taskutil"
	"github.com/puppetlabs/nebula-tasks/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type TestGetSpec struct {
	Name    string   `json:"name"`
	SomeNum int      `json:"someNum"`
	Data    []string `json:"data"`
}

func TestGetOutput(t *testing.T) {
	testSpec := &TestGetSpec{
		Name:    "test1",
		SomeNum: 12,
		Data:    []string{"something", "else"},
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

		output, _ := task.ReadData("{.name}")
		require.Equal(t, testSpec.Name, string(output))

		output, _ = task.ReadData("{.someNum}")
		require.Equal(t, strconv.Itoa(testSpec.SomeNum), string(output))

		output, _ = task.ReadData("{.data[0]}")
		require.Equal(t, testSpec.Data[0], string(output))

		output, _ = task.ReadData("{.data[1]}")
		require.Equal(t, testSpec.Data[1], string(output))
	}, opts)
}
