package op

import (
	"context"
	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
)

type StateManager interface {
	Get(ctx context.Context, stepName, key string) (*outputs.Output, errors.Error)
}

type EncodeDecodingStateManager struct {
	delegate StateManager
}

func (m EncodeDecodingStateManager) Get(ctx context.Context, stepName, key string) (*outputs.Output, errors.Error) {
	out, err := m.delegate.Get(ctx, stepName, key)
	if err != nil {
		return nil, err
	}

	decoded, derr := transfer.DecodeFromTransfer(out.Value)
	if derr != nil {
		return nil, errors.NewOutputsValueDecodingError().WithCause(derr)
	}

	out.Value = string(decoded)

	return out, nil
}

func NewEncodeDecodingStateManager(sm StateManager) *EncodeDecodingStateManager {
	return &EncodeDecodingStateManager{delegate: sm}
}
