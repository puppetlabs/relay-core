package authenticate

import "context"

// Raw is unprocessed token information, such as a compact JWT.
type Raw []byte

// An Intermediary looks up a token from a source environment.
type Intermediary interface {
	Next(ctx context.Context, state *Authentication) (Raw, error)
}

var _ Intermediary = Raw(nil)

func (r Raw) Next(ctx context.Context, state *Authentication) (Raw, error) {
	return r, nil
}

type IntermediaryFunc func(ctx context.Context, state *Authentication) (Raw, error)

var _ Intermediary = IntermediaryFunc(nil)

func (ifn IntermediaryFunc) Next(ctx context.Context, state *Authentication) (Raw, error) {
	return ifn(ctx, state)
}

type ChainIntermediaryFunc func(ctx context.Context, prev Raw) (Intermediary, error)

type ChainIntermediary struct {
	initial Intermediary
	fns     []ChainIntermediaryFunc
}

var _ Intermediary = &ChainIntermediary{}

func (ci *ChainIntermediary) Next(ctx context.Context, state *Authentication) (Raw, error) {
	raw, err := ci.initial.Next(ctx, state)
	if err != nil {
		return nil, err
	}

	for _, fn := range ci.fns {
		next, err := fn(ctx, raw)
		if err != nil {
			return nil, err
		}

		raw, err = next.Next(ctx, state)
		if err != nil {
			return nil, err
		}
	}

	return raw, nil
}

func NewChainIntermediary(initial Intermediary, fns ...ChainIntermediaryFunc) *ChainIntermediary {
	return &ChainIntermediary{
		initial: initial,
		fns:     append([]ChainIntermediaryFunc{}, fns...),
	}
}
