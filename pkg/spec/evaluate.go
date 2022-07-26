package spec

import (
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	"github.com/puppetlabs/leg/relspec/pkg/relspec"
)

func newMappingTypeResolvers(opts *Options) map[string]relspec.MappingTypeResolver[*References] {
	return map[string]relspec.MappingTypeResolver[*References]{
		"Answer":     &AnswerMappingTypeResolver{AnswerTypeResolver: opts.AnswerTypeResolver},
		"Data":       DataMappingTypeResolver(opts.DataTypeResolvers),
		"Connection": &ConnectionMappingTypeResolver{ConnectionTypeResolver: opts.ConnectionTypeResolver},
		"Output":     &OutputMappingTypeResolver{OutputTypeResolver: opts.OutputTypeResolver},
		"Parameter":  &ParameterMappingTypeResolver{ParameterTypeResolver: opts.ParameterTypeResolver},
		"Secret":     &SecretMappingTypeResolver{SecretTypeResolver: opts.SecretTypeResolver},
	}
}

func newTemplateEnvironment(opts *Options) map[string]evaluate.Expandable[*References] {
	env := make(map[string]evaluate.Expandable[*References])

	// Data gets loaded first so names can't override.
	for name, resolver := range opts.DataTypeResolvers {
		// We can't use the default resolver. It must be named.
		if name == "" {
			continue
		}

		env[name] = &DataTemplateEnvironment{DataTypeResolver: resolver, Name: name}
	}

	env["connections"] = &ConnectionTemplateEnvironment{ConnectionTypeResolver: opts.ConnectionTypeResolver}
	env["outputs"] = &OutputTemplateEnvironment{OutputTypeResolver: opts.OutputTypeResolver}
	env["parameters"] = &ParameterTemplateEnvironment{ParameterTypeResolver: opts.ParameterTypeResolver}
	env["secrets"] = &SecretTemplateEnvironment{SecretTypeResolver: opts.SecretTypeResolver}
	env["steps"] = &StatusTemplateEnvironment{StatusTypeResolver: opts.StatusTypeResolver}

	return env
}

func NewEvaluator(opts ...Option) evaluate.Evaluator[*References] {
	o := &Options{
		DataTypeResolvers:      map[string]DataTypeResolver{},
		SecretTypeResolver:     NoOpSecretTypeResolver,
		ConnectionTypeResolver: NoOpConnectionTypeResolver,
		OutputTypeResolver:     NoOpOutputTypeResolver,
		ParameterTypeResolver:  NoOpParameterTypeResolver,
		AnswerTypeResolver:     NoOpAnswerTypeResolver,
		StatusTypeResolver:     NoOpStatusTypeResolver,
	}
	o.ApplyOptions(opts)

	return relspec.NewEvaluator[*References](
		relspec.WithMappingTypeResolvers[*References](newMappingTypeResolvers(o)),
		relspec.WithTemplateEnvironment[*References](newTemplateEnvironment(o)),
	)
}
