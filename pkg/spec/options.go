package spec

type Options struct {
	DataTypeResolvers      map[string]DataTypeResolver
	SecretTypeResolver     SecretTypeResolver
	ConnectionTypeResolver ConnectionTypeResolver
	OutputTypeResolver     OutputTypeResolver
	ParameterTypeResolver  ParameterTypeResolver
	AnswerTypeResolver     AnswerTypeResolver
	StatusTypeResolver     StatusTypeResolver
}

type Option interface {
	ApplyToOptions(target *Options)
}

func (o *Options) ApplyOptions(opts []Option) {
	for _, opt := range opts {
		opt.ApplyToOptions(o)
	}
}

type WithDataTypeResolver struct {
	Name             string
	Default          bool
	DataTypeResolver DataTypeResolver
}

var _ Option = WithDataTypeResolver{}

func (wdtr WithDataTypeResolver) ApplyToOptions(target *Options) {
	target.DataTypeResolvers[wdtr.Name] = wdtr.DataTypeResolver
	if wdtr.Default {
		target.DataTypeResolvers[""] = wdtr.DataTypeResolver
	}
}

type WithSecretTypeResolver struct{ SecretTypeResolver }

var _ Option = WithSecretTypeResolver{}

func (wstr WithSecretTypeResolver) ApplyToOptions(target *Options) {
	target.SecretTypeResolver = wstr.SecretTypeResolver
}

type WithConnectionTypeResolver struct{ ConnectionTypeResolver }

var _ Option = WithConnectionTypeResolver{}

func (wctr WithConnectionTypeResolver) ApplyToOptions(target *Options) {
	target.ConnectionTypeResolver = wctr.ConnectionTypeResolver
}

type WithOutputTypeResolver struct{ OutputTypeResolver }

var _ Option = WithOutputTypeResolver{}

func (wotr WithOutputTypeResolver) ApplyToOptions(target *Options) {
	target.OutputTypeResolver = wotr.OutputTypeResolver
}

type WithParameterTypeResolver struct{ ParameterTypeResolver }

var _ Option = WithParameterTypeResolver{}

func (wptr WithParameterTypeResolver) ApplyToOptions(target *Options) {
	target.ParameterTypeResolver = wptr.ParameterTypeResolver
}

type WithAnswerTypeResolver struct{ AnswerTypeResolver }

var _ Option = WithAnswerTypeResolver{}

func (watr WithAnswerTypeResolver) ApplyToOptions(target *Options) {
	target.AnswerTypeResolver = watr.AnswerTypeResolver
}

type WithStatusTypeResolver struct{ StatusTypeResolver }

var _ Option = WithStatusTypeResolver{}

func (wstr WithStatusTypeResolver) ApplyToOptions(target *Options) {
	target.StatusTypeResolver = wstr.StatusTypeResolver
}
