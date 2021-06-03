package resolve

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/expr/model"
)

type noOpDataTypeResolver struct{}

func (*noOpDataTypeResolver) ResolveData(ctx context.Context) (interface{}, error) {
	return nil, &model.DataNotFoundError{}
}

var (
	NoOpDataTypeResolver       DataTypeResolver       = &noOpDataTypeResolver{}
	NoOpSecretTypeResolver     SecretTypeResolver     = NewMemorySecretTypeResolver(map[string]string{})
	NoOpConnectionTypeResolver ConnectionTypeResolver = NewMemoryConnectionTypeResolver(map[MemoryConnectionKey]interface{}{})
	NoOpOutputTypeResolver     OutputTypeResolver     = NewMemoryOutputTypeResolver(map[MemoryOutputKey]interface{}{})
	NoOpParameterTypeResolver  ParameterTypeResolver  = NewMemoryParameterTypeResolver(map[string]interface{}{})
	NoOpAnswerTypeResolver     AnswerTypeResolver     = NewMemoryAnswerTypeResolver(map[MemoryAnswerKey]interface{}{})
)
