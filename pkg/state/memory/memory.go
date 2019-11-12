package memory

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sync"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
)

type StateManager struct {
	data map[string]map[string][]byte

	sync.Mutex
}

func (sm *StateManager) Get(ctx context.Context, stepName, key string) (*outputs.Output, errors.Error) {
	sm.Lock()
	defer sm.Unlock()

	stepHash := sha1.Sum([]byte(stepName))
	stepHashKey := hex.EncodeToString(stepHash[:])

	if sm.data == nil {
		return nil, errors.NewOutputsTaskNotFound(stepName)
	}

	if sm.data[stepHashKey] == nil {
		return nil, errors.NewOutputsTaskNotFound(stepName)
	}

	if sm.data[stepHashKey][key] == nil {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	return &outputs.Output{
		TaskName: stepName,
		Key:      key,
		Value:    string(sm.data[stepHashKey][key]),
	}, nil
}

func New() *StateManager {
	return &StateManager{}
}
