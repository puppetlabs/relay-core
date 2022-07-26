package spec_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/puppetlabs/leg/encoding/transfer"
	"github.com/puppetlabs/relay-core/pkg/spec"
	"github.com/stretchr/testify/require"
)

func jsonSecret(name string) map[string]any {
	return map[string]any{"$type": "Secret", "name": name}
}

func jsonConnection(connectionType, name string) map[string]any {
	return map[string]any{"$type": "Connection", "type": connectionType, "name": name}
}

func jsonOutput(from, name string) map[string]any {
	return map[string]any{"$type": "Output", "from": from, "name": name}
}

func jsonParameter(name string) map[string]any {
	return map[string]any{"$type": "Parameter", "name": name}
}

func jsonAnswer(askRef, name string) map[string]any {
	return map[string]any{"$type": "Answer", "askRef": askRef, "name": name}
}

func jsonInvocation(name string, args any) map[string]any {
	return map[string]any{fmt.Sprintf("$fn.%s", name): args}
}

func jsonEncoding(ty transfer.EncodingType, data any) map[string]any {
	return map[string]any{"$encoding": string(ty), "data": data}
}

func TestJSON(t *testing.T) {
	expected := []byte(`{
		"x": {"$fn.concat": [{"$type": "Parameter", "name": "foo"}, "bar"]},
		"y": {"$fn.concat": ["a", "b"]}
	}`)

	var tree spec.JSONTree
	require.NoError(t, json.Unmarshal(expected, &tree))

	actual, err := json.Marshal(tree)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))
}

func TestJSONEncodedUnsafeString(t *testing.T) {
	expected := []byte(`{
		"foo": {
			"$encoding": "base64",
			"data": "SGVsbG8sIJCiikU="
		}
	}`)

	var tree spec.JSONTree
	require.NoError(t, json.Unmarshal(expected, &tree))

	actual, err := json.Marshal(tree)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(actual))
}
