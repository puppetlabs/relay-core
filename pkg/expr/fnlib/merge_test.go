package fnlib_test

import (
	"context"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/stretchr/testify/require"
)

func TestMerge(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("merge")
	require.NoError(t, err)

	tt := []struct {
		Name          string
		Req           func() (fn.Invoker, error)
		Expected      interface{}
		ExpectedError string
	}{
		{
			Name: "positional",
			Req: func() (fn.Invoker, error) {
				return desc.PositionalInvoker([]interface{}{
					map[string]interface{}{"a": "b", "c": map[string]interface{}{"d": "e"}},
					map[string]interface{}{"a": "overwritten", "c": map[string]interface{}{"f": "added"}},
				})
			},
			Expected: map[string]interface{}{
				"a": "overwritten",
				"c": map[string]interface{}{"d": "e", "f": "added"},
			},
		},
		{
			Name: "keyword deep",
			Req: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]interface{}{
					"mode": "deep",
					"objects": []interface{}{
						map[string]interface{}{"a": "b", "c": map[string]interface{}{"d": "e"}},
						map[string]interface{}{"a": "overwritten", "c": map[string]interface{}{"f": "added"}},
					},
				})
			},
			Expected: map[string]interface{}{
				"a": "overwritten",
				"c": map[string]interface{}{"d": "e", "f": "added"},
			},
		},
		{
			Name: "keyword shallow",
			Req: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]interface{}{
					"mode": "shallow",
					"objects": []interface{}{
						map[string]interface{}{"a": "b", "c": map[string]interface{}{"d": "e"}},
						map[string]interface{}{"a": "overwritten", "c": map[string]interface{}{"f": "overwritten"}},
					},
				})
			},
			Expected: map[string]interface{}{
				"a": "overwritten",
				"c": map[string]interface{}{"f": "overwritten"},
			},
		},
		{
			Name: "invalid mode",
			Req: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]interface{}{
					"mode":    "secret",
					"objects": []interface{}{},
				})
			},
			ExpectedError: `fn: arg "mode": unexpected value "secret", wanted one of "deep" or "shallow"`,
		},
		{
			Name: "merge candidate is not a map",
			Req: func() (fn.Invoker, error) {
				return desc.PositionalInvoker([]interface{}{
					map[string]interface{}{"a": "b", "c": map[string]interface{}{"d": "e"}},
					"hi",
				})
			},
			ExpectedError: `fn: arg 1: array index 1: fn: unexpected type string (wanted map[string]interface {})`,
		},
	}
	for _, test := range tt {
		t.Run(test.Name, func(t *testing.T) {
			invoker, err := test.Req()
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			if test.ExpectedError != "" {
				require.EqualError(t, err, test.ExpectedError)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, test.Expected, r)
		})
	}
}
