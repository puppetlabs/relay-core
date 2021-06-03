package model

import "context"

type Connection struct {
	Type       string
	Name       string
	Attributes map[string]interface{}
}

type ConnectionManager interface {
	List(ctx context.Context) ([]*Connection, error)
	Get(ctx context.Context, typ, name string) (*Connection, error)
}
