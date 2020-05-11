package authenticate

import "context"

// A Validator provides additional assertions that a resolver's claims are
// valid.
type Validator interface {
	Validate(ctx context.Context, claims *Claims) (bool, error)
}

type ValidatorFunc func(ctx context.Context, claims *Claims) (bool, error)

var _ Validator = ValidatorFunc(nil)

func (vf ValidatorFunc) Validate(ctx context.Context, claims *Claims) (bool, error) {
	return vf(ctx, claims)
}
