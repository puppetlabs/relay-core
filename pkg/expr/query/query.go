package query

import (
	"context"

	"github.com/PaesslerAG/gval"
	"github.com/puppetlabs/leg/jsonutil/pkg/jsonpath"
	jsonpathtemplate "github.com/puppetlabs/leg/jsonutil/pkg/jsonpath/template"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/parse"
	"github.com/puppetlabs/relay-core/pkg/expr/pathlang"
)

type Language func(ctx context.Context, ev model.Evaluator, u *model.Unresolvable) gval.Language

func PathLanguage(opts ...pathlang.Option) Language {
	o := &pathlang.Options{}
	o.ApplyOptions(opts)

	return func(ctx context.Context, ev model.Evaluator, u *model.Unresolvable) gval.Language {
		return pathlang.NewFactory(o, pathlang.WithEvaluator{Evaluator: ev}).Expression(u)
	}
}

func PathTemplateLanguage(opts ...pathlang.Option) Language {
	o := &pathlang.Options{}
	o.ApplyOptions(opts)

	return func(ctx context.Context, ev model.Evaluator, u *model.Unresolvable) gval.Language {
		return pathlang.NewFactory(o, pathlang.WithEvaluator{Evaluator: ev}).Template(u)
	}
}

var (
	JSONPathLanguage Language = func(ctx context.Context, ev model.Evaluator, u *model.Unresolvable) gval.Language {
		return gval.NewLanguage(
			jsonpathtemplate.ExpressionLanguage(),
			gval.VariableSelector(jsonpath.VariableSelector(variableVisitor(ev, u))),
		)
	}

	JSONPathTemplateLanguage Language = func(ctx context.Context, ev model.Evaluator, u *model.Unresolvable) gval.Language {
		return jsonpathtemplate.TemplateLanguage(
			jsonpathtemplate.WithExpressionLanguageVariableVisitor(variableVisitor(ev, u)),
			jsonpathtemplate.WithFormatter(func(v interface{}) (string, error) {
				rv, err := model.EvaluateAll(ctx, ev, v)
				if err != nil {
					return "", err
				} else if !rv.Complete() {
					u.Extends(rv.Unresolvable)
				} else {
					v = rv.Value
				}

				return jsonpathtemplate.DefaultFormatter(v)
			}),
		)
	}
)

func EvaluateQuery(ctx context.Context, ev model.Evaluator, lang Language, tree parse.Tree, query string) (*model.Result, error) {
	var u model.Unresolvable

	path, err := lang(ctx, ev, &u).NewEvaluable(query)
	if err != nil {
		return nil, err
	}

	v, err := path(ctx, tree)
	if err != nil {
		if u.AsError() != nil {
			// It is vastly likely that an error that occurs during traversal
			// with unresolvable data is caused by that missing data as opposed
			// to a problem with the query itself. Either way, we can't evaluate
			// the query correctly until the data is fixed.
			return &model.Result{Unresolvable: u}, nil
		}

		return nil, err
	}

	r, err := model.EvaluateAll(ctx, ev, v)
	if err != nil {
		return nil, err
	}

	// Add any other unresolved paths in here (provided by the variable selector).
	r.Unresolvable.Extends(u)
	return r, nil
}
