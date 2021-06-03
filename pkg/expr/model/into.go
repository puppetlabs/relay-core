package model

import (
	"context"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
)

func mapstructureHookFunc(ctx context.Context, ev Evaluator, u *Unresolvable) mapstructure.DecodeHookFunc {
	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		depth := -1

		// Copy so we can potentially use the zero value below.
		check := to
		for check.Kind() == reflect.Ptr {
			check = check.Elem()
		}

		if check.Kind() == reflect.Struct {
			// We only evaluate one level of nesting for structs, because their
			// children will get correctly traversed once the data exists.
			depth = 1
		}

		r, err := ev.Evaluate(ctx, data, depth)
		if err != nil {
			return nil, err
		} else if !r.Complete() {
			u.Extends(r.Unresolvable)

			// We return the zero value of the type to eliminate confusion.
			return reflect.Zero(to).Interface(), nil
		}

		return r.Value, nil
	}
}

func EvaluateInto(ctx context.Context, ev Evaluator, from parse.Tree, to interface{}) (Unresolvable, error) {
	var u Unresolvable

	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructureHookFunc(ctx, ev, &u),
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToTimeHookFunc(time.RFC3339Nano),
		),
		ZeroFields: true,
		Result:     to,
		TagName:    "spec",
	})
	if err != nil {
		return u, err
	}

	return u, d.Decode(from)
}
