package pathlang

import (
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type WithEvaluator struct{ model.Evaluator }

var _ Option = WithEvaluator{}

func (we WithEvaluator) ApplyToOptions(target *Options) {
	target.Evaluator = we.Evaluator
}

type WithFunctionMap struct{ fn.Map }

var _ Option = WithFunctionMap{}

func (wfm WithFunctionMap) ApplyToOptions(target *Options) {
	target.FunctionMap = wfm.Map
}
