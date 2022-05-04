package evaluate

import (
	"github.com/puppetlabs/relay-core/pkg/expr/fn"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
)

type WithFunctionMap struct{ fn.Map }

var _ Option = WithFunctionMap{}

func (wfm WithFunctionMap) ApplyToOptions(target *Options) {
	target.FunctionMap = wfm.Map
}

type WithDataTypeResolver struct {
	Name             string
	Default          bool
	DataTypeResolver resolve.DataTypeResolver
}

var _ Option = WithDataTypeResolver{}

func (wdtr WithDataTypeResolver) ApplyToOptions(target *Options) {
	target.DataTypeResolvers[wdtr.Name] = wdtr.DataTypeResolver
	if wdtr.Default {
		target.DataTypeResolvers[""] = wdtr.DataTypeResolver
	}
}

type WithSecretTypeResolver struct{ resolve.SecretTypeResolver }

var _ Option = WithSecretTypeResolver{}

func (wstr WithSecretTypeResolver) ApplyToOptions(target *Options) {
	target.SecretTypeResolver = wstr.SecretTypeResolver
}

type WithConnectionTypeResolver struct{ resolve.ConnectionTypeResolver }

var _ Option = WithConnectionTypeResolver{}

func (wctr WithConnectionTypeResolver) ApplyToOptions(target *Options) {
	target.ConnectionTypeResolver = wctr.ConnectionTypeResolver
}

type WithOutputTypeResolver struct{ resolve.OutputTypeResolver }

var _ Option = WithOutputTypeResolver{}

func (wotr WithOutputTypeResolver) ApplyToOptions(target *Options) {
	target.OutputTypeResolver = wotr.OutputTypeResolver
}

type WithParameterTypeResolver struct{ resolve.ParameterTypeResolver }

var _ Option = WithParameterTypeResolver{}

func (wptr WithParameterTypeResolver) ApplyToOptions(target *Options) {
	target.ParameterTypeResolver = wptr.ParameterTypeResolver
}

type WithAnswerTypeResolver struct{ resolve.AnswerTypeResolver }

var _ Option = WithAnswerTypeResolver{}

func (watr WithAnswerTypeResolver) ApplyToOptions(target *Options) {
	target.AnswerTypeResolver = watr.AnswerTypeResolver
}

type WithStatusTypeResolver struct{ resolve.StatusTypeResolver }

var _ Option = WithStatusTypeResolver{}

func (wstr WithStatusTypeResolver) ApplyToOptions(target *Options) {
	target.StatusTypeResolver = wstr.StatusTypeResolver
}
