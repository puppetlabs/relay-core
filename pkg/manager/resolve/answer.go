package resolve

import (
	"context"

	exprmodel "github.com/puppetlabs/relay-core/pkg/expr/model"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/model"
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
		return nil, &exprmodel.AnswerNotFoundError{AskRef: askRef, Name: name}
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
