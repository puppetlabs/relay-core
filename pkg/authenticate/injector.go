package authenticate

import "context"

// An Injector runs after validation is complete to provide additional context
// to consuming applications.
type Injector interface {
	Inject(ctx context.Context, claims *Claims) error
}

type InjectorFunc func(ctx context.Context, claims *Claims) error

var _ Injector = InjectorFunc(nil)

func (ij InjectorFunc) Inject(ctx context.Context, claims *Claims) error {
	return ij(ctx, claims)
}
