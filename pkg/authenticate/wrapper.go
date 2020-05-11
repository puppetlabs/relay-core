package authenticate

import "context"

type Wrapper interface {
	Wrap(ctx context.Context, raw Raw) (Raw, error)
}

type WrapperFunc func(ctx context.Context, raw Raw) (Raw, error)

var _ Wrapper = WrapperFunc(nil)

func (wf WrapperFunc) Wrap(ctx context.Context, raw Raw) (Raw, error) {
	return wf(ctx, raw)
}

type ChainWrapper struct {
	delegates []Wrapper
}

var _ Wrapper = &ChainWrapper{}

func (cw *ChainWrapper) Wrap(ctx context.Context, raw Raw) (Raw, error) {
	var err error

	for _, delegate := range cw.delegates {
		raw, err = delegate.Wrap(ctx, raw)
		if err != nil {
			return nil, err
		}
	}

	return raw, nil
}

func NewChainWrapper(delegates ...Wrapper) *ChainWrapper {
	return &ChainWrapper{
		delegates: delegates,
	}
}

type WrappedIssuer struct {
	delegate Issuer
	wrapper  Wrapper
}

var _ Issuer = &WrappedIssuer{}

func (wi *WrappedIssuer) Issue(ctx context.Context, claims *Claims) (Raw, error) {
	raw, err := wi.delegate.Issue(ctx, claims)
	if err != nil {
		return nil, err
	}

	return wi.wrapper.Wrap(ctx, raw)
}

func NewWrappedIssuer(delegate Issuer, wrapper Wrapper) *WrappedIssuer {
	return &WrappedIssuer{
		delegate: delegate,
		wrapper:  wrapper,
	}
}
