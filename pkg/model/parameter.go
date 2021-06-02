package model

import "context"

type Parameter struct {
	Name  string
	Value interface{}
}

type ParameterGetterManager interface {
	List(ctx context.Context) ([]*Parameter, error)
	Get(ctx context.Context, name string) (*Parameter, error)
}

type ParameterSetterManager interface {
	Set(ctx context.Context, name string, value interface{}) (*Parameter, error)
}

type ParameterManager interface {
	ParameterGetterManager
	ParameterSetterManager
}
