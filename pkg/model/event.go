package model

import "context"

type Event struct {
	Data map[string]interface{}
}

type EventManager interface {
	Emit(ctx context.Context, data map[string]interface{}) (*Event, error)
}
