package model

import "context"

type Expandable interface {
	Expand(ctx context.Context, depth int) (*Result, error)
}

type staticExpandable struct {
	r *Result
}

func (se *staticExpandable) Expand(ctx context.Context, depth int) (*Result, error) {
	return se.r, nil
}

func StaticExpandable(v interface{}, u Unresolvable) Expandable {
	return &staticExpandable{r: &Result{Value: v, Unresolvable: u}}
}
