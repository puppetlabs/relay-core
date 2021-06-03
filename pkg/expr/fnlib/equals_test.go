package fnlib_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestConditionals(t *testing.T) {
	equals, err := fnlib.Library().Descriptor("equals")
	require.NoError(t, err)

	notEquals, err := fnlib.Library().Descriptor("notEquals")
	require.NoError(t, err)

	cases := []struct {
		descriptor     fn.Descriptor
		args           []interface{}
		expectedResult bool
		expectedError  error
	}{
		{
			descriptor:     equals,
			args:           []interface{}{"foobar", "foobar"},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{10, 10},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{10.5, 10.5},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{[]interface{}{"foo", "bar"}, []interface{}{"foo", "bar"}},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{[]interface{}{1, 2}, []interface{}{1, 2}},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{[]interface{}{1.1, 2.0}, []interface{}{1.1, 2.0}},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{[]interface{}{1.1, 2.0}, []interface{}{1.1, 2.0}},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{map[string]interface{}{"foo": "bar"}, map[string]interface{}{"foo": "bar"}},
			expectedResult: true,
		},
		{
			descriptor:     equals,
			args:           []interface{}{"true", true},
			expectedResult: false,
		},
		{
			descriptor:     equals,
			args:           []interface{}{"10", 10},
			expectedResult: false,
		},
		{
			descriptor:     equals,
			args:           []interface{}{10.5, 10},
			expectedResult: false,
		},
		{
			descriptor:     equals,
			args:           []interface{}{1, 2},
			expectedResult: false,
		},
		{
			descriptor:    equals,
			args:          []interface{}{1, 2, 3},
			expectedError: &fn.ArityError{Wanted: []int{2}, Got: 3},
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{"foobar", "barfoo"},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{10, 50},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{10.0, 50.5},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{[]interface{}{"foo", "bar", "baz"}, []interface{}{"foo", "bar"}},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{[]interface{}{1, 2, 3}, []interface{}{1, 2}},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{[]interface{}{1.1, 2.0, 3.2}, []interface{}{1.1, 2.0}},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{[]interface{}{1.1, 2.0, 3.2}, []interface{}{1.1, 2.0}},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{map[string]interface{}{"foo": "bar", "baz": "biz"}, map[string]interface{}{"foo": "bar"}},
			expectedResult: true,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{true, true},
			expectedResult: false,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{10, 10},
			expectedResult: false,
		},
		{
			descriptor:     notEquals,
			args:           []interface{}{"foobar", "foobar"},
			expectedResult: false,
		},
		{
			descriptor:    notEquals,
			args:          []interface{}{1, 2, 3},
			expectedError: &fn.ArityError{Wanted: []int{2}, Got: 3},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d %v", i, c.args), func(t *testing.T) {
			invoker, err := c.descriptor.PositionalInvoker(model.DefaultEvaluator, c.args)
			if c.expectedError != nil {
				require.EqualError(t, err, c.expectedError.Error())
			} else {
				require.NoError(t, err)

				r, err := invoker.Invoke(context.Background())
				require.NoError(t, err)

				require.True(t, r.Complete())
				require.Equal(t, c.expectedResult, r.Value)
			}
		})
	}
}
