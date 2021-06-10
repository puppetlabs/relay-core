package model

import "context"

type Secret struct {
	Name, Value string
}

type SecretManager interface {
	List(ctx context.Context) ([]*Secret, error)
	Get(ctx context.Context, name string) (*Secret, error)
}
