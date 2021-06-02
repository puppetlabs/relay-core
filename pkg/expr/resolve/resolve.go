package resolve

import (
	"context"
)

type DataTypeResolver interface {
	ResolveData(ctx context.Context) (interface{}, error)
}

type SecretTypeResolver interface {
	ResolveAllSecrets(ctx context.Context) (map[string]string, error)
	ResolveSecret(ctx context.Context, name string) (string, error)
}

type ConnectionTypeResolver interface {
	ResolveAllConnections(ctx context.Context) (map[string]map[string]interface{}, error)
	ResolveTypeOfConnections(ctx context.Context, connectionType string) (map[string]interface{}, error)
	ResolveConnection(ctx context.Context, connectionType, name string) (interface{}, error)
}

type OutputTypeResolver interface {
	ResolveAllOutputs(ctx context.Context) (map[string]map[string]interface{}, error)
	ResolveStepOutputs(ctx context.Context, from string) (map[string]interface{}, error)
	ResolveOutput(ctx context.Context, from, name string) (interface{}, error)
}

type ParameterTypeResolver interface {
	ResolveAllParameters(ctx context.Context) (map[string]interface{}, error)
	ResolveParameter(ctx context.Context, name string) (interface{}, error)
}

type AnswerTypeResolver interface {
	ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error)
}
