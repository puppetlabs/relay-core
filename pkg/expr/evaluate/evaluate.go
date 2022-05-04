package evaluate

import (
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/fnlib"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
)

type Options struct {
	FunctionMap            fn.Map
	DataTypeResolvers      map[string]resolve.DataTypeResolver
	SecretTypeResolver     resolve.SecretTypeResolver
	ConnectionTypeResolver resolve.ConnectionTypeResolver
	OutputTypeResolver     resolve.OutputTypeResolver
	ParameterTypeResolver  resolve.ParameterTypeResolver
	AnswerTypeResolver     resolve.AnswerTypeResolver
	StatusTypeResolver     resolve.StatusTypeResolver
}

type Option interface {
	ApplyToOptions(target *Options)
}

func (o *Options) ApplyOptions(opts []Option) {
	for _, opt := range opts {
		opt.ApplyToOptions(o)
	}
}

func NewEvaluator(opts ...Option) model.Evaluator {
	o := &Options{
		FunctionMap:            fnlib.Library(),
		DataTypeResolvers:      map[string]resolve.DataTypeResolver{},
		SecretTypeResolver:     resolve.NoOpSecretTypeResolver,
		ConnectionTypeResolver: resolve.NoOpConnectionTypeResolver,
		OutputTypeResolver:     resolve.NoOpOutputTypeResolver,
		ParameterTypeResolver:  resolve.NoOpParameterTypeResolver,
		AnswerTypeResolver:     resolve.NoOpAnswerTypeResolver,
		StatusTypeResolver:     resolve.NoOpStatusTypeResolver,
	}
	o.ApplyOptions(opts)

	return model.NewEvaluator(&model.VisitorFuncs{
		VisitMapFunc:    evaluateMap(o),
		VisitStringFunc: evaluateTemplate(o),
	})
}
