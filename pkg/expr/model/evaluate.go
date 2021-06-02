package model

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"github.com/puppetlabs/relay-core/pkg/expr/parse"
)

type Evaluator interface {
	Evaluate(ctx context.Context, tree parse.Tree, depth int) (*Result, error)
}

type Visitor interface {
	VisitExpandable(ctx context.Context, ex Expandable, depth int, next Evaluator) (*Result, error)
	VisitSlice(ctx context.Context, s []interface{}, depth int, next Evaluator) (*Result, error)
	VisitMap(ctx context.Context, m map[string]interface{}, depth int, next Evaluator) (*Result, error)
	VisitString(ctx context.Context, s string, depth int, next Evaluator) (*Result, error)
}

type VisitorFuncs struct {
	VisitExpandableFunc func(ctx context.Context, ex Expandable, depth int, next Evaluator) (*Result, error)
	VisitSliceFunc      func(ctx context.Context, s []interface{}, depth int, next Evaluator) (*Result, error)
	VisitMapFunc        func(ctx context.Context, m map[string]interface{}, depth int, next Evaluator) (*Result, error)
	VisitStringFunc     func(ctx context.Context, s string, depth int, next Evaluator) (*Result, error)
}

func (vfs *VisitorFuncs) VisitExpandable(ctx context.Context, ex Expandable, depth int, next Evaluator) (*Result, error) {
	if vfs.VisitExpandableFunc != nil {
		return vfs.VisitExpandableFunc(ctx, ex, depth, next)
	}

	return ex.Expand(ctx, depth)
}

func (vfs *VisitorFuncs) VisitSlice(ctx context.Context, s []interface{}, depth int, next Evaluator) (*Result, error) {
	if vfs.VisitSliceFunc != nil {
		return vfs.VisitSliceFunc(ctx, s, depth, next)
	}

	if depth == 1 {
		return &Result{Value: s}, nil
	}

	r := &Result{}
	l := make([]interface{}, len(s))
	for i, v := range s {
		nv, err := next.Evaluate(ctx, v, depth-1)
		if err != nil {
			return nil, &PathEvaluationError{
				Path:  strconv.Itoa(i),
				Cause: err,
			}
		}

		r.Extends(nv)
		l[i] = nv.Value
	}

	r.Value = l
	return r, nil
}

func (vfs *VisitorFuncs) VisitMap(ctx context.Context, m map[string]interface{}, depth int, next Evaluator) (*Result, error) {
	if vfs.VisitMapFunc != nil {
		return vfs.VisitMapFunc(ctx, m, depth, next)
	}

	if depth == 1 {
		return &Result{Value: m}, nil
	}

	r := &Result{}
	rm := make(map[string]interface{}, len(m))
	for k, v := range m {
		nv, err := next.Evaluate(ctx, v, depth-1)
		if err != nil {
			return nil, &PathEvaluationError{Path: k, Cause: err}
		}

		r.Extends(nv)
		rm[k] = nv.Value
	}

	r.Value = rm
	return r, nil
}

func (vfs *VisitorFuncs) VisitString(ctx context.Context, s string, depth int, next Evaluator) (*Result, error) {
	if vfs.VisitStringFunc != nil {
		return vfs.VisitStringFunc(ctx, s, depth, next)
	}

	return &Result{Value: s}, nil
}

var DefaultVisitor Visitor = &VisitorFuncs{}

type visitorEvaluator struct {
	visitor Visitor
}

func (ve *visitorEvaluator) evaluate(ctx context.Context, tree parse.Tree, depth int) (*Result, error) {
	if depth == 0 {
		return &Result{Value: tree}, nil
	}

	switch vt := tree.(type) {
	case Expandable:
		return ve.visitor.VisitExpandable(ctx, vt, depth, ve)
	case []interface{}:
		return ve.visitor.VisitSlice(ctx, vt, depth, ve)
	case map[string]interface{}:
		return ve.visitor.VisitMap(ctx, vt, depth, ve)
	case string:
		return ve.visitor.VisitString(ctx, vt, depth, ve)
	default:
		return &Result{Value: tree}, nil
	}
}

func (ve *visitorEvaluator) Evaluate(ctx context.Context, tree parse.Tree, depth int) (*Result, error) {
	candidate, err := ve.evaluate(ctx, tree, depth)
	if err != nil {
		return nil, err
	}

	switch candidate.Value.(type) {
	// Valid JSON types per https://golang.org/pkg/encoding/json/:
	case bool, float64, string, []interface{}, map[string]interface{}, nil:
		return candidate, nil
	// We support a set of additional YAML scalar(-ish) types decoded by
	// gopkg.in/yaml.v3.
	case []byte, int, int64, uint, uint64, time.Time:
		return candidate, nil
	default:
		return nil, &UnsupportedValueError{Type: reflect.TypeOf(candidate.Value)}
	}
}

func NewEvaluator(visitor Visitor) Evaluator {
	return &visitorEvaluator{visitor: visitor}
}

var DefaultEvaluator = NewEvaluator(DefaultVisitor)

func EvaluateAll(ctx context.Context, ev Evaluator, tree parse.Tree) (*Result, error) {
	return ev.Evaluate(ctx, tree, -1)
}

func EvaluateAllSlice(ctx context.Context, ev Evaluator, s []interface{}) ([]*Result, error) {
	rs := make([]*Result, len(s))
	for i, v := range s {
		r, err := EvaluateAll(ctx, ev, v)
		if err != nil {
			return nil, err
		}

		rs[i] = r
	}
	return rs, nil
}

func EvaluateAllMap(ctx context.Context, ev Evaluator, m map[string]interface{}) (map[string]*Result, error) {
	rm := make(map[string]*Result, len(m))
	for key, v := range m {
		r, err := EvaluateAll(ctx, ev, v)
		if err != nil {
			return nil, err
		}

		rm[key] = r
	}
	return rm, nil
}
