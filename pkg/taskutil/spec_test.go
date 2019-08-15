package taskutil

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/testutil"
	"github.com/stretchr/testify/require"
)

type TestSpec struct {
	Name    string `json:"name"`
	SomeNum int    `json:"someNum"`
}

func TestDefaultSpecPlan(t *testing.T) {
	opts := testutil.MockMetadataAPIOptions{
		Name: "test1",
		ResponseObject: TestSpec{
			Name:    "test1",
			SomeNum: 12,
		},
	}

	testutil.WithMockMetadataAPI(t, func(ts *httptest.Server) {
		testSpec := TestSpec{}

		opts := DefaultPlanOptions{
			Client:  ts.Client(),
			SpecURL: fmt.Sprintf("%s/specs/test1", ts.URL),
		}

		require.NoError(t, PopulateSpecFromDefaultPlan(&testSpec, opts))
		require.Equal(t, "test1", testSpec.Name)
		require.Equal(t, 12, testSpec.SomeNum)
	}, opts)
}
