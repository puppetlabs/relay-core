package model

import (
	"context"

	"github.com/PaesslerAG/gval"
)

type Expandable interface {
	Expand(ctx context.Context, depth int) (*Result, error)
}

type staticExpandable struct {
	r *Result
}

var _ Expandable = &staticExpandable{}

func (se *staticExpandable) Expand(ctx context.Context, depth int) (*Result, error) {
	return se.r, nil
}

func StaticExpandable(v interface{}, u Unresolvable) Expandable {
	return &staticExpandable{r: &Result{Value: v, Unresolvable: u}}
}

type evalExpandable struct {
	eval      gval.Evaluable
	parameter interface{}
}

var _ Expandable = &evalExpandable{}

func (ee *evalExpandable) Expand(ctx context.Context, depth int) (*Result, error) {
	ctx, u := ContextWithNewUnresolvable(ctx)

	v, err := ee.eval(ctx, ee.parameter)
	if err != nil {
		return nil, err
	}

	return &Result{
		Value:        v,
		Unresolvable: *u,
	}, nil
}

func EvalExpandable(eval gval.Evaluable, parameter interface{}) Expandable {
	return &evalExpandable{
		eval:      eval,
		parameter: parameter,
	}
}
