package model

import (
	"context"

	gval "github.com/puppetlabs/paesslerag-gval"
)

type Evaluable interface {
	Evaluate(ctx context.Context, depth int) (*Result, error)
	EvaluateAll(ctx context.Context) (*Result, error)
	EvaluateQuery(ctx context.Context, query string) (*Result, error)
}

type staticEvaluable struct {
	value interface{}
}

var _ Evaluable = &staticEvaluable{}

func (se *staticEvaluable) Evaluate(ctx context.Context, depth int) (*Result, error) {
	return &Result{Value: se.value}, nil
}

func (se *staticEvaluable) EvaluateAll(ctx context.Context) (*Result, error) {
	return se.Evaluate(ctx, -1)
}

func (se *staticEvaluable) EvaluateQuery(ctx context.Context, query string) (*Result, error) {
	pl, err := gval.NewLanguage(gval.Base()).NewEvaluable(query)
	if err != nil {
		return nil, err
	}

	v, err := pl(ctx, se.value)
	if err != nil {
		return nil, err
	}

	return &Result{Value: v}, nil
}

func StaticEvaluable(value interface{}) Evaluable {
	return &staticEvaluable{value: value}
}

type unresolvableEvaluable struct {
	u Unresolvable
}

var _ Evaluable = &unresolvableEvaluable{}

func (ue *unresolvableEvaluable) Evaluate(ctx context.Context, depth int) (*Result, error) {
	return &Result{Unresolvable: ue.u}, nil
}

func (ue *unresolvableEvaluable) EvaluateAll(ctx context.Context) (*Result, error) {
	return ue.Evaluate(ctx, -1)
}

func (ue *unresolvableEvaluable) EvaluateQuery(Ctx context.Context, query string) (*Result, error) {
	return &Result{Unresolvable: ue.u}, nil
}

func UnresolvableEvaluable(u Unresolvable) Evaluable {
	return &unresolvableEvaluable{u: u}
}

func EvaluateAllSlice(ctx context.Context, s []Evaluable) ([]*Result, error) {
	rs := make([]*Result, len(s))
	for i, ev := range s {
		r, err := ev.EvaluateAll(ctx)
		if err != nil {
			return nil, err
		}

		rs[i] = r
	}
	return rs, nil
}

func EvaluateAllMap(ctx context.Context, m map[string]Evaluable) (map[string]*Result, error) {
	rm := make(map[string]*Result, len(m))
	for key, ev := range m {
		r, err := ev.EvaluateAll(ctx)
		if err != nil {
			return nil, err
		}

		rm[key] = r
	}
	return rm, nil
}
