package pathlang

import (
	"context"
	"text/scanner"

	"github.com/PaesslerAG/gval"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type evalExpandable struct {
	eval      gval.Evaluable
	parameter interface{}
}

var _ model.Expandable = &evalExpandable{}

func (ee *evalExpandable) Expand(ctx context.Context, depth int) (*model.Result, error) {
	ctx, u := model.ContextWithNewUnresolvable(ctx)

	v, err := ee.eval(ctx, ee.parameter)
	if err != nil {
		return nil, err
	}

	return &model.Result{
		Value:        v,
		Unresolvable: *u,
	}, nil
}

func evalIdentFuncKeyword(o *Options, name string, args map[string]gval.Evaluable) gval.Evaluable {
	return func(ctx context.Context, parameter interface{}) (interface{}, error) {
		descriptor, err := o.FunctionMap.Descriptor(name)
		if err != nil {
			return model.StaticExpandable(nil, model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{{Name: name, Cause: err}},
			}), nil
		}

		m := make(map[string]interface{}, len(args))
		for k, eval := range args {
			m[k] = &evalExpandable{
				eval:      eval,
				parameter: parameter,
			}
		}

		invoker, err := descriptor.KeywordInvoker(o.Evaluator, m)
		if err != nil {
			return model.StaticExpandable(nil, model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{{Name: name, Cause: err}},
			}), nil
		}

		r, err := invoker.Invoke(ctx)
		if err != nil {
			return nil, &model.InvocationError{Name: name, Cause: err}
		} else if !r.Complete() {
			return model.StaticExpandable(nil, r.Unresolvable), nil
		}

		return r.Value, nil
	}
}

func evalIdentFuncPositional(o *Options, name string, args []gval.Evaluable) gval.Evaluable {
	return func(ctx context.Context, parameter interface{}) (interface{}, error) {
		descriptor, err := o.FunctionMap.Descriptor(name)
		if err != nil {
			return model.StaticExpandable(nil, model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{{Name: name, Cause: err}},
			}), nil
		}

		l := make([]interface{}, len(args))
		for i, eval := range args {
			l[i] = &evalExpandable{
				eval:      eval,
				parameter: parameter,
			}
		}

		invoker, err := descriptor.PositionalInvoker(o.Evaluator, l)
		if err != nil {
			return model.StaticExpandable(nil, model.Unresolvable{
				Invocations: []model.UnresolvableInvocation{{Name: name, Cause: err}},
			}), nil
		}

		r, err := invoker.Invoke(ctx)
		if err != nil {
			return nil, &model.InvocationError{Name: name, Cause: err}
		} else if !r.Complete() {
			return model.StaticExpandable(nil, r.Unresolvable), nil
		}

		return r.Value, nil
	}
}

func identFuncKeyword(ctx context.Context, p *gval.Parser, o *Options, name string, args map[string]gval.Evaluable) (gval.Evaluable, error) {
	if p.Scan() != scanner.Ident {
		return nil, p.Expected("function argument name", scanner.Ident)
	}

	key := p.TokenText()

	if p.Scan() != ':' {
		return nil, p.Expected("function argument name", ':')
	}

	arg, err := p.ParseExpression(ctx)
	if err != nil {
		return nil, err
	}

	if args == nil {
		args = make(map[string]gval.Evaluable)
	}

	args[key] = arg

	switch p.Scan() {
	case ')':
		return evalIdentFuncKeyword(o, name, args), nil
	case ',':
		// Another arg.
		return identFuncKeyword(ctx, p, o, name, args)
	default:
		return nil, p.Expected("function argument", ',', ')')
	}
}

func identFuncPositional(ctx context.Context, p *gval.Parser, o *Options, name string, args []gval.Evaluable) (gval.Evaluable, error) {
	arg, err := p.ParseExpression(ctx)
	if err != nil {
		return nil, err
	}

	args = append(args, arg)

	switch p.Scan() {
	case ')':
		return evalIdentFuncPositional(o, name, args), nil
	case ',':
		// Another arg.
		return identFuncPositional(ctx, p, o, name, args)
	default:
		return nil, p.Expected("function argument", ',', ')')
	}
}

func identFunc(ctx context.Context, p *gval.Parser, o *Options, name string) (gval.Evaluable, error) {
	switch p.Scan() {
	case ')':
		// Positional invocation with no arguments.
		return evalIdentFuncPositional(o, name, nil), nil
	case scanner.Ident:
		candidate := p.TokenText()
		switch p.Peek() {
		case ':':
			p.Scan() // == ':'

			arg, err := p.ParseExpression(ctx)
			if err != nil {
				return nil, err
			}

			switch p.Scan() {
			case ',':
				// Keyword invocation with several arguments.
				return identFuncKeyword(ctx, p, o, name, map[string]gval.Evaluable{candidate: arg})
			case ')':
				// Keyword invocation with a single argument.
				return evalIdentFuncKeyword(o, name, map[string]gval.Evaluable{candidate: arg}), nil
			default:
				return nil, p.Expected("function argument", ',', ')')
			}
		default:
			// Assume positional invocation with data argument.
			p.Camouflage("function argument", ':', ',', ')')
			arg, err := p.ParseExpression(ctx)
			if err != nil {
				return nil, err
			}

			switch p.Scan() {
			case ',':
				return identFuncPositional(ctx, p, o, name, []gval.Evaluable{arg})
			case ')':
				return evalIdentFuncPositional(o, name, []gval.Evaluable{arg}), nil
			default:
				return nil, p.Expected("function argument", ',', ')')
			}
		}
	default:
		// Must be positional invocation.
		p.Camouflage("function argument", ':', ',', ')')
		return identFuncPositional(ctx, p, o, name, nil)
	}
}

func identVar(ctx context.Context, p *gval.Parser, vars []gval.Evaluable) (gval.Evaluable, error) {
	switch p.Scan() {
	case '.':
		switch p.Scan() {
		case scanner.Ident:
			return identVar(ctx, p, append(vars, p.Const(p.TokenText())))
		default:
			return nil, p.Expected("field", scanner.Ident)
		}
	case '[':
		key, err := p.ParseExpression(ctx)
		if err != nil {
			return nil, err
		}

		switch p.Scan() {
		case ']':
			return identVar(ctx, p, append(vars, key))
		default:
			return nil, p.Expected("key", ']')
		}
	default:
		p.Camouflage("variable", '.', '[')
		return p.Var(vars...), nil
	}
}

func ident(o *Options) gval.Language {
	return gval.NewLanguage(
		gval.PrefixMetaPrefix(scanner.Ident, func(ctx context.Context, p *gval.Parser) (call string, alternative func() (gval.Evaluable, error), err error) {
			call = p.TokenText()
			alternative = func() (gval.Evaluable, error) {
				switch p.Scan() {
				case '(':
					return identFunc(ctx, p, o, call)
				default:
					p.Camouflage("identifier", '.', '[', '(')
					return identVar(ctx, p, []gval.Evaluable{p.Const(call)})
				}
			}
			return
		}),
	)
}
