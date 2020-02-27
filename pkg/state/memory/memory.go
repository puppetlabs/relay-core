package memory

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
)

type StateManager struct {
	data map[string]string

	sync.Mutex
}

func (sm *StateManager) Get(ctx context.Context, taskHash [sha1.Size]byte, key string) (*state.State, errors.Error) {
	sm.Lock()
	defer sm.Unlock()

	name := fmt.Sprintf("task-%x-state", taskHash)
	taskHashKey := hex.EncodeToString(taskHash[:])

	if sm.data == nil {
		return nil, errors.NewStateTaskNotFound(name)
	}

	if _, ok := sm.data[taskHashKey]; !ok {
		return nil, errors.NewStateTaskNotFound(name)
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

func (sm *StateManager) Set(ctx context.Context, taskHash [sha1.Size]byte, value io.Reader) errors.Error {
	sm.Lock()
	defer sm.Unlock()

	taskHashKey := hex.EncodeToString(taskHash[:])

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
