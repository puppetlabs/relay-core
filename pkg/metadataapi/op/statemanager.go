package op

import (
	"bytes"
	"context"
	"io"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type StateManager interface {
	Get(ctx context.Context, metadata *task.Metadata, key string) (*state.State, errors.Error)
	Set(ctx context.Context, metadata *task.Metadata, value io.Reader) errors.Error
}

type EncodeDecodingStateManager struct {
	delegate StateManager
}

func (m EncodeDecodingStateManager) Get(ctx context.Context, metadata *task.Metadata, key string) (*state.State, errors.Error) {
	out, err := m.delegate.Get(ctx, metadata, key)
	if err != nil {
		return nil, err
	}

	decoded, derr := transfer.DecodeFromTransfer(out.Value)
	if derr != nil {
		return nil, errors.NewStateValueDecodingError().WithCause(derr)
	}

	out.Value = string(decoded)

	return out, nil
}

func (m EncodeDecodingStateManager) Set(ctx context.Context, metadata *task.Metadata, value io.Reader) errors.Error {
	buf := &bytes.Buffer{}

	_, err := buf.ReadFrom(value)
	if err != nil {
		return errors.NewStateValueReadError().WithCause(err)
	}

	encoded, err := transfer.EncodeForTransfer(buf.Bytes())
	if err != nil {
		return errors.NewStateValueEncodingError().WithCause(err).Bug()
	}

	buf = bytes.NewBufferString(encoded)

	return m.delegate.Set(ctx, metadata, buf)
}

func NewEncodeDecodingStateManager(sm StateManager) *EncodeDecodingStateManager {
	return &EncodeDecodingStateManager{delegate: sm}
}
