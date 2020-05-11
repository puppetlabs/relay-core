package model

import "context"

type State struct {
	Name  string
	Value interface{}
}

type StateGetterManager interface {
	Get(ctx context.Context, name string) (*State, error)
}

type StateSetterManager interface {
	Set(ctx context.Context, name string, value interface{}) (*State, error)
}

type StateManager interface {
	StateGetterManager
	StateSetterManager
}
