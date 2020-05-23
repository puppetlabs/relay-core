package authenticate

import (
	"context"
)

// Authentication is the current authentication state. Intermediaries and
// resolvers can add new validators and injectors to the state.
type Authentication struct {
	validators *[]Validator
	injectors  *[]Injector
}

func (a *Authentication) AddValidator(v Validator) {
	*a.validators = append(*a.validators, v)
}

func (a *Authentication) AddInjector(i Injector) {
	*a.injectors = append(*a.injectors, i)
}

func NewInitializedAuthentication(validators *[]Validator, injectors *[]Injector) *Authentication {
	return &Authentication{
		validators: validators,
		injectors:  injectors,
	}
}

func NewAuthentication() *Authentication {
	return NewInitializedAuthentication(&[]Validator{}, &[]Injector{})
}

// Authenticator provides client authentication using a token. It resolves and
// validates claims, finally injecting contextual information as needed.
type Authenticator struct {
	intermediary Intermediary
	resolver     Resolver
	validators   []Validator
	injectors    []Injector
}

func (a *Authenticator) Authenticate(ctx context.Context) (bool, error) {
	validators := append([]Validator{}, a.validators...)
	injectors := append([]Injector{}, a.injectors...)

	state := NewInitializedAuthentication(&validators, &injectors)

	raw, err := a.intermediary.Next(ctx, state)
	if _, ok := err.(*NotFoundError); ok {
		log(ctx).Warn("authentication failed in intermediary", "error", err)
		return false, nil
	} else if err != nil {
		return false, err
	}

	claims, err := a.resolver.Resolve(ctx, state, raw)
	if _, ok := err.(*NotFoundError); ok {
		log(ctx).Warn("authentication failed in resolver", "error", err)
		return false, nil
	} else if err != nil {
		return false, err
	}

	// Always validate action for a claim. A malformed claim that cannot
	// recreate an action exactly should not be accepted.
	if claims.Action() == nil {
		log(ctx).Warn("authentication failed because claim did not contain a valid action reference")
		return false, nil
	}

	for _, validator := range validators {
		if ok, err := validator.Validate(ctx, claims); err != nil || !ok {
			return false, err
		}
	}

	for _, injector := range injectors {
		if err := injector.Inject(ctx, claims); err != nil {
			return false, err
		}
	}

	return true, nil
}

type AuthenticatorOption func(a *Authenticator)

func AuthenticatorWithValidator(v Validator) AuthenticatorOption {
	return func(a *Authenticator) {
		a.validators = append(a.validators, v)
	}
}

func AuthenticatorWithInjector(i Injector) AuthenticatorOption {
	return func(a *Authenticator) {
		a.injectors = append(a.injectors, i)
	}
}

func NewAuthenticator(intermediary Intermediary, resolver Resolver, opts ...AuthenticatorOption) *Authenticator {
	a := &Authenticator{
		intermediary: intermediary,
		resolver:     resolver,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}
