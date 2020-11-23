package fnlib

import (
	"context"
	"strings"

	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var concatDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string { return "Concatenates string arguments into a single string" },
	PositionalInvokerFunc: func(args []model.Evaluable) (fn.Invoker, error) {
		if len(args) == 0 {
			return fn.StaticInvoker(""), nil
		}

		fn := fn.EvaluatedPositionalInvoker(args, func(ctx context.Context, args []interface{}) (m interface{}, err error) {
			strs := make([]string, len(args))
			for i, iarg := range args {
				arg, err := toString(iarg)
				if err != nil {
					return nil, &fn.PositionalArgError{
						Arg:   i + 1,
						Cause: err,
					}
				}

				strs[i] = arg
			}

			return strings.Join(strs, ""), nil
		})
		return fn, nil
	},
}
