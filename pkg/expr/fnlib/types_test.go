package fnlib_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestToString(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("toString")
	require.NoError(t, err)

	tests := []struct {
		Name          string
		Value         interface{}
		Expected      interface{}
		ExpectedError error
	}{
		{
			Name:     "string",
			Value:    "foo",
			Expected: "foo",
		},
		{
			Name:     "bytes",
			Value:    []byte("foo"),
			Expected: "foo",
		},
		{
			Name:     "boolean",
			Value:    true,
			Expected: "true",
		},
		{
			Name:     "null",
			Expected: "",
		},
		{
			Name:     "integer",
			Value:    42,
			Expected: "42",
		},
		{
			Name:     "decimal",
			Value:    42.424242,
			Expected: "42.424242",
		},
		{
			Name:     "time",
			Value:    time.Unix(0, 0).In(time.UTC),
			Expected: "1970-01-01T00:00:00Z",
		},
		{
			Name:  "invalid",
			Value: map[string]interface{}{"foo": "bar"},
			ExpectedError: &fn.PositionalArgError{
				Arg: 1,
				Cause: &fn.UnexpectedTypeError{
					Wanted: []reflect.Type{
						reflect.TypeOf(nil),
						reflect.TypeOf(""),
						reflect.TypeOf([]byte(nil)),
						reflect.TypeOf(time.Time{}),
						reflect.TypeOf(int(0)),
						reflect.TypeOf(int64(0)),
						reflect.TypeOf(float64(0)),
						reflect.TypeOf(false),
					},
					Got: reflect.TypeOf(map[string]interface{}(nil)),
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			invoker, err := desc.PositionalInvoker(model.DefaultEvaluator, []interface{}{test.Value})
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			if test.ExpectedError != nil {
				require.Equal(t, test.ExpectedError, err)
			} else {
				require.NoError(t, err)
				require.True(t, r.Complete())
				require.Equal(t, test.Expected, r.Value)
			}
		})
	}
}
