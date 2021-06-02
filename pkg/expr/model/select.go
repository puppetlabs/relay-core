package model

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
	"github.com/puppetlabs/leg/gvalutil/pkg/eval"
)

func VariableSelector(ev Evaluator, u *Unresolvable) func(path gval.Evaluables) gval.Evaluable {
	return func(path gval.Evaluables) gval.Evaluable {
		return func(ctx context.Context, v interface{}) (rv interface{}, err error) {
			var parents []interface{}
			defer func() {
				if err != nil {
					for i := len(parents) - 1; i >= 0; i-- {
						err = &PathEvaluationError{
							Path:  fmt.Sprintf("%v", parents[i]),
							Cause: err,
						}
					}
				}
			}()

			cv := v

			for _, key := range path {
				key, err := key(ctx, cv)
				if err != nil {
					return nil, err
				}
				parents = append(parents, key)

				switch vt := v.(type) {
				case eval.Indexable:
					v, err = vt.Index(ctx, key)
					if err != nil {
						return nil, err
					}
				default:
					nr, err := ev.Evaluate(ctx, vt, 1)
					if err != nil {
						return nil, err
					} else if !nr.Complete() {
						u.Extends(nr.Unresolvable)
						return nil, nil
					}

					v, err = eval.Select(ctx, nr.Value, key)
					if err != nil {
						return nil, err
					}
				}
			}

			// TODO: This is potentially resource-intensive in places where it
			// isn't needed (e.g., unused arguments to coalesce()), but right
			// now we sadly don't have a good workaround that also allows
			// operators to work correctly.
			nr, err := EvaluateAll(ctx, ev, v)
			if err != nil {
				return nil, err
			} else if !nr.Complete() {
				u.Extends(nr.Unresolvable)
				// Note: no return here; we'll use the expanded value even
				// if it's unresolvable.
			}

			return nr.Value, nil
		}
	}
}
