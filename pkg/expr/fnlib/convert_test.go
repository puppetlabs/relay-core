package fnlib_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/expr/convert"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/stretchr/testify/require"
)

func TestConvertMarkdown(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("convertMarkdown")
	require.NoError(t, err)

	tcs := []struct {
		Name        string
		ConvertType convert.ConvertType
		Markdown    string
		Expected    string
	}{
		{
			Name:        "Sample monitor event",
			ConvertType: convert.ConvertTypeJira,
			Markdown:    "%%% @contact [![imageTitle](imageUrl)](imageRedirect) `data{context} > threshold` Detailed description. - - - [[linkTitle1](link1)] · [[linkTitle2](link2)] %%%",
			Expected:    "@contact \n\n[!imageUrl!|imageRedirect] {code}data{context} > threshold{code} Detailed description.\n----\n[[linkTitle1|link1]] · [[linkTitle2|link2]]",
		},
	}

	for _, test := range tcs {
		t.Run(fmt.Sprintf("%s", test.Name), func(t *testing.T) {
			invoker, err := desc.PositionalInvoker([]model.Evaluable{
				model.StaticEvaluable(test.ConvertType.String()),
				model.StaticEvaluable(test.Markdown),
			})
			require.NoError(t, err)

			r, err := invoker.Invoke(context.Background())
			require.NoError(t, err)

			require.True(t, r.Complete())
			require.Equal(t, test.Expected, r.Value)

			invoker, err = desc.KeywordInvoker(map[string]model.Evaluable{
				"to":      model.StaticEvaluable(test.ConvertType.String()),
				"content": model.StaticEvaluable(test.Markdown),
			})

			r, err = invoker.Invoke(context.Background())
			require.NoError(t, err)

			require.True(t, r.Complete())
			require.Equal(t, test.Expected, r.Value)
		})
	}
}

func TestConvertMarkdownFunction(t *testing.T) {
	desc, err := fnlib.Library().Descriptor("convertMarkdown")
	require.NoError(t, err)

	tcs := []struct {
		Name                 string
		Invoker              func() (fn.Invoker, error)
		ExpectedInvokeError  error
		ExpectedInvokerError error
	}{
		{
			Name: "keyword invoker with unsupported convert type",
			Invoker: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]model.Evaluable{
					"to":      model.StaticEvaluable("foo"),
					"content": model.StaticEvaluable("bar"),
				})
			},
			ExpectedInvokerError: convert.ErrConvertTypeNotSupported,
		},
		{
			Name: "keyword invoker with invalid to keyword type",
			Invoker: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]model.Evaluable{
					"to":      model.StaticEvaluable(false),
					"content": model.StaticEvaluable("bar"),
				})
			},
			ExpectedInvokerError: &fn.KeywordArgError{
				Arg: "to",
				Cause: &fn.UnexpectedTypeError{
					Wanted: []reflect.Type{
						reflect.TypeOf(""),
					},
					Got: reflect.TypeOf(false),
				},
			},
		},
		{
			Name: "keyword invoker with invalid content keyword type",
			Invoker: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]model.Evaluable{
					"to":      model.StaticEvaluable("jira"),
					"content": model.StaticEvaluable(false),
				})
			},
			ExpectedInvokerError: &fn.KeywordArgError{
				Arg: "content",
				Cause: &fn.UnexpectedTypeError{
					Wanted: []reflect.Type{
						reflect.TypeOf(""),
					},
					Got: reflect.TypeOf(false),
				},
			},
		},
		{
			Name: "keyword invoker with missing to keyword",
			Invoker: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]model.Evaluable{
					"content": model.StaticEvaluable("bar"),
				})
			},
			ExpectedInvokeError: &fn.KeywordArgError{
				Arg:   "to",
				Cause: fn.ErrArgNotFound,
			},
		},
		{
			Name: "keyword invoker with missing content",
			Invoker: func() (fn.Invoker, error) {
				return desc.KeywordInvoker(map[string]model.Evaluable{
					"to": model.StaticEvaluable("jira"),
				})
			},
			ExpectedInvokeError: &fn.KeywordArgError{
				Arg:   "content",
				Cause: fn.ErrArgNotFound,
			},
		},
	}

	for _, test := range tcs {
		t.Run(fmt.Sprintf("%s", test.Name), func(t *testing.T) {
			invoker, err := test.Invoker()
			if test.ExpectedInvokeError != nil {
				require.Equal(t, test.ExpectedInvokeError, err)
			} else {
				require.NoError(t, err)

				_, err = invoker.Invoke(context.Background())
				if test.ExpectedInvokerError != nil {
					require.Equal(t, test.ExpectedInvokerError, err)
				} else {
					require.NoError(t, err)
				}
			}
		})
	}
}
