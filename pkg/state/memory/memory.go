package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type StateManager struct {
	data map[string]string

	sync.Mutex
}

func (sm *StateManager) Get(ctx context.Context, metadata *task.Metadata, key string) (*state.State, errors.Error) {
	sm.Lock()
	defer sm.Unlock()

	taskHashKey := metadata.Hash.HexEncoding()

	if sm.data == nil {
		return nil, errors.NewStateTaskNotFound(taskHashKey)
	}

	if _, ok := sm.data[taskHashKey]; !ok {
		return nil, errors.NewStateTaskNotFound(taskHashKey)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(sm.data[taskHashKey]), &data); err != nil {
		return nil, errors.NewStateValueDecodingError().WithCause(err)
	}

	val, ok := data[key]
	if !ok {
		return nil, errors.NewStateKeyNotFound(key)
	}

	return &state.State{
		Key:   key,
		Value: val.(string),
	}, nil
}

func (sm *StateManager) Set(ctx context.Context, metadata *task.Metadata, value io.Reader) errors.Error {
	sm.Lock()
	defer sm.Unlock()

	taskHashKey := metadata.Hash.HexEncoding()

	if sm.data == nil {
		sm.data = make(map[string]string)
	}

	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(value)
	if err != nil {
		return errors.NewStateValueReadError().WithCause(err)
	}

	sm.data[taskHashKey] = buf.String()

	return nil
}

func New() *StateManager {
	return &StateManager{}
}
