package evaluate

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
)

type ScopedEvaluator struct {
	parent *Evaluator
	tree   parse.Tree
}

func (se *ScopedEvaluator) Evaluate(ctx context.Context, depth int) (*model.Result, error) {
	return se.parent.Evaluate(ctx, se.tree, depth)
}

func (se *ScopedEvaluator) EvaluateAll(ctx context.Context) (*model.Result, error) {
	return se.parent.EvaluateAll(ctx, se.tree)
}

func (se *ScopedEvaluator) EvaluateInto(ctx context.Context, target interface{}) (model.Unresolvable, error) {
	return se.parent.EvaluateInto(ctx, se.tree, target)
}

func (se *ScopedEvaluator) EvaluateQuery(ctx context.Context, query string) (*model.Result, error) {
	return se.parent.EvaluateQuery(ctx, se.tree, query)
}

func (se *ScopedEvaluator) Copy(opts ...Option) *ScopedEvaluator {
	return &ScopedEvaluator{parent: se.parent.Copy(opts...), tree: se.tree}
}

func NewScopedEvaluator(obj parse.Tree, opts ...Option) *ScopedEvaluator {
	return NewEvaluator(opts...).ScopeTo(obj)
}
