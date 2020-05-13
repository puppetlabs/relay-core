package model

import "context"

type Secret struct {
	Name, Value string
}

type SecretManager interface {
	Get(ctx context.Context, name string) (*Secret, error)
}
