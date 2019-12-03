package memory

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"sync"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/state"
)

type StateManager struct {
	data map[string]map[string][]byte

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

	if sm.data[taskHashKey] == nil {
		return nil, errors.NewStateTaskNotFound(name)
	}

	if sm.data[taskHashKey][key] == nil {
		return nil, errors.NewStateKeyNotFound(key)
	}

	return &state.State{
		Key:   key,
		Value: string(sm.data[taskHashKey][key]),
	}, nil
}

func (sm *StateManager) Set(ctx context.Context, taskHash [sha1.Size]byte, key string, value io.Reader) errors.Error {
	sm.Lock()
	defer sm.Unlock()

	taskHashKey := hex.EncodeToString(taskHash[:])

	if key == "" {
		return errors.NewStateKeyEmptyError()
	}

	if sm.data == nil {
		sm.data = make(map[string]map[string][]byte)
	}

	if sm.data[taskHashKey] == nil {
		sm.data[taskHashKey] = make(map[string][]byte)
	}

	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(value)
	if err != nil {
		return errors.NewStateValueReadError().WithCause(err)
	}

	sm.data[taskHashKey][key] = buf.Bytes()

	return nil
}

func New() *StateManager {
	return &StateManager{}
}
