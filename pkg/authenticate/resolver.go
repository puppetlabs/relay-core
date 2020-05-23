package authenticate

import (
	"context"
	"fmt"
)

// A Resolver finds the claims associated with a token.
type Resolver interface {
	Resolve(ctx context.Context, state *Authentication, raw Raw) (*Claims, error)
}

type ResolverFunc func(ctx context.Context, state *Authentication, raw Raw) (*Claims, error)

var _ Resolver = ResolverFunc(nil)

func (rf ResolverFunc) Resolve(ctx context.Context, state *Authentication, raw Raw) (*Claims, error) {
	return rf(ctx, state, raw)
}

// AnyResolver picks the first resolver that resolves claims successfully. If a
// resolver returns an error other than ErrNotFound, it is immediately
// propagated.
type AnyResolver struct {
	delegates []Resolver
}

var _ Resolver = &AnyResolver{}

func (ar *AnyResolver) Resolve(ctx context.Context, state *Authentication, raw Raw) (*Claims, error) {
	// Fast case for one delegate.
	if len(ar.delegates) == 1 {
		return ar.delegates[0].Resolve(ctx, state, raw)
	}

	var causes []error

	for _, delegate := range ar.delegates {
		// Temporary state that will be propagated back on success.
		var validators []Validator
		var injectors []Injector
		ts := NewInitializedAuthentication(&validators, &injectors)

		claims, err := delegate.Resolve(ctx, ts, raw)
		if _, ok := err.(*NotFoundError); ok {
			causes = append(causes, err)
			continue
		} else if err != nil {
			return nil, err
		}

		for _, validator := range validators {
			state.AddValidator(validator)
		}

		for _, injector := range injectors {
			state.AddInjector(injector)
		}

		return claims, nil
	}

	return nil, &NotFoundError{
		Reason: fmt.Sprintf("none of the %d available resolvers authenticated this token", len(ar.delegates)),
		Causes: causes,
	}
}

func NewAnyResolver(delegates []Resolver) *AnyResolver {
	return &AnyResolver{
		delegates: delegates,
	}
}
