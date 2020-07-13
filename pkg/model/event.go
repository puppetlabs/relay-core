package model

import "context"

type Event struct {
	Data map[string]interface{}
	Key  string
}

type EventManager interface {
	Emit(ctx context.Context, data map[string]interface{}, key string) (*Event, error)
}
