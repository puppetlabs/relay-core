package serialize_test

import (
	"encoding/json"
	"testing"

	"github.com/puppetlabs/nebula-tasks/pkg/expr/serialize"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	expected := []byte(`{
		"x": {"$fn.concat": [{"$type": "Parameter", "name": "foo"}, "bar"]},
		"y": {"$fn.concat": ["a", "b"]}
	}`)

	var tree serialize.JSONTree
	require.NoError(t, json.Unmarshal(expected, &tree))

	actual, err := json.Marshal(tree)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))
}
