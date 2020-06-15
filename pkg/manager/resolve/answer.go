package resolve

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/expr/resolve"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
)

// TODO
// ====
//
// - This needs to be renamed to Query/Reply.
// - The askRef here is not handled yet.

type AnswerTypeResolver struct {
	m model.StateGetterManager
}

var _ resolve.AnswerTypeResolver = &AnswerTypeResolver{}

func (atr *AnswerTypeResolver) ResolveAnswer(ctx context.Context, askRef, name string) (interface{}, error) {
	s, err := atr.m.Get(ctx, name)
	if err == model.ErrNotFound {
		return nil, &resolve.AnswerNotFoundError{AskRef: askRef, Name: name}
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
