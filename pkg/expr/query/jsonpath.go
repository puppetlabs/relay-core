package query

import (
	"context"

	"github.com/puppetlabs/leg/jsonutil/pkg/jsonpath"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

func variableVisitor(ev model.Evaluator, u *model.Unresolvable) jsonpath.VariableVisitor {
	return jsonpath.VariableVisitorFuncs{
		VisitChildFunc: func(ctx context.Context, parameter interface{}, key interface{}, next func(ctx context.Context, pv jsonpath.PathValue) error) error {
			// We need to evaluate the base parameter before indexing in. This
			// is because the base parameter could be itself a $type, $encoding,
			// etc.
			nr, err := ev.Evaluate(ctx, parameter, 1)
			if err != nil {
				return err
			} else if !nr.Complete() {
				u.Extends(nr.Unresolvable)
				return nil
			}

			return jsonpath.DefaultVariableVisitor().VisitChild(ctx, nr.Value, key, func(ctx context.Context, pv jsonpath.PathValue) error {
				// Expand just this value without recursing.
				nr, err := ev.Evaluate(ctx, pv.Value, 1)
				if err != nil {
					return err
				} else if !nr.Complete() {
					u.Extends(nr.Unresolvable)
					return nil
				}

				pv.Value = nr.Value
				return next(ctx, pv)
			})
		},
	}
}
