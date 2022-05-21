package specadapter

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/spec"
)

// TODO
// ====
//
// - This needs to be renamed to Query/Reply.
// - The askRef here is not handled yet.

type AnswerTypeResolver struct {
	m model.StateGetterManager
}

var _ spec.AnswerTypeResolver = &AnswerTypeResolver{}

func (atr *AnswerTypeResolver) ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error) {
	s, err := atr.m.Get(ctx, name)
	if err == model.ErrNotFound {
		return nil, spec.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return s.Value, nil
}

func NewAnswerTypeResolver(m model.StateGetterManager) *AnswerTypeResolver {
	return &AnswerTypeResolver{
		m: m,
	}
}
