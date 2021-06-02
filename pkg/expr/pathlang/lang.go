package pathlang

import (
	"context"

	"github.com/PaesslerAG/gval"
	"github.com/generikvault/gvalstrings"
	"github.com/puppetlabs/leg/gvalutil/pkg/template"
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type Options struct {
	Evaluator   model.Evaluator
	FunctionMap fn.Map
}

type Option interface {
	ApplyToOptions(target *Options)
}

var _ Option = &Options{}

func (o *Options) ApplyToOptions(target *Options) {
	if o.Evaluator != nil {
		target.Evaluator = o.Evaluator
	}
	if o.FunctionMap != nil {
		target.FunctionMap = o.FunctionMap
	}
}

func (o *Options) ApplyOptions(opts []Option) {
	for _, opt := range opts {
		opt.ApplyToOptions(o)
	}
}

type Factory struct {
	opts *Options
}

func (f *Factory) Expression(u *model.Unresolvable) gval.Language {
	return gval.NewLanguage(
		base,
		ident(f.opts),
		gval.VariableSelector(model.VariableSelector(f.opts.Evaluator, u)),
	)
}

func (f *Factory) Template(u *model.Unresolvable) gval.Language {
	return template.Language(
		template.WithJoiner{
			Joiner: template.NewStringJoiner(template.WithEmptyStringsEliminated(true)),
		},
		template.WithDelimitedLanguage{
			DelimitedLanguage: &template.DelimitedLanguage{
				Start:    "${",
				End:      "}",
				Language: f.Expression(u),
			},
		},
	)
}

func NewFactory(opts ...Option) *Factory {
	o := &Options{
		Evaluator:   model.DefaultEvaluator,
		FunctionMap: fn.NewMap(map[string]fn.Descriptor{}),
	}
	o.ApplyOptions(opts)

	return &Factory{opts: o}
}

var DefaultFactory = NewFactory()

var base = gval.NewLanguage(
	gval.Base(),
	gval.Arithmetic(),
	gval.Bitmask(),
	gval.Text(),
	gval.PropositionalLogic(),
	gval.JSON(),
	gvalstrings.SingleQuoted(),
	gval.PrefixExtension('$', func(ctx context.Context, p *gval.Parser) (gval.Evaluable, error) {
		switch p.Scan() {
		case '.':
			return p.ParseSublanguage(ctx, gval.Ident())
		default:
			p.Camouflage("variable", '.')
			return p.Var(), nil
		}
	}),
	gval.PostfixOperator("|>", func(c context.Context, p *gval.Parser, pre gval.Evaluable) (gval.Evaluable, error) {
		post, err := p.ParseExpression(c)
		if err != nil {
			return nil, err
		}
		return func(c context.Context, v interface{}) (interface{}, error) {
			v, err := pre(c, v)
			if err != nil {
				return nil, err
			}
			return post(c, v)
		}, nil
	}),
)
