package model

import "context"

type Connection struct {
	Type       string
	Name       string
	Attributes map[string]string
}

type ConnectionManager interface {
	Get(ctx context.Context, typ, name string) (*Connection, error)
}
