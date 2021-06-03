package fnlib

import (
	"context"

	"github.com/puppetlabs/leg/timeutil/pkg/clockctx"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

var nowDescriptor = fn.DescriptorFuncs{
	DescriptionFunc: func() string { return "Retrieves the current system time in UTC" },
	PositionalInvokerFunc: func(ev model.Evaluator, args []interface{}) (fn.Invoker, error) {
		if len(args) != 0 {
			return nil, &fn.ArityError{Wanted: []int{0}, Got: len(args)}
		}

		fn := fn.EvaluatedPositionalInvoker(ev, args, func(ctx context.Context, args []interface{}) (interface{}, error) {
			return clockctx.Clock(ctx).Now(), nil
		})
		return fn, nil
	},
}
