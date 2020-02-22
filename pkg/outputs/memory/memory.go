package memory

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"sync"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"
	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
)

type OutputsManager struct {
	data map[string]map[string]transfer.JSONInterface

	sync.Mutex
}

func (om *OutputsManager) Get(ctx context.Context, taskName, key string) (*outputs.Output, errors.Error) {
	om.Lock()
	defer om.Unlock()

	taskHash := sha1.Sum([]byte(taskName))
	taskHashKey := hex.EncodeToString(taskHash[:])

	if om.data == nil {
		return nil, errors.NewOutputsTaskNotFound(taskName)
	}

	if om.data[taskHashKey] == nil {
		return nil, errors.NewOutputsTaskNotFound(taskName)
	}

	value, ok := om.data[taskHashKey][key]
	if !ok {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	return &outputs.Output{
		TaskName: taskName,
		Key:      key,
		Value:    value,
	}, nil
}

func (om *OutputsManager) Put(ctx context.Context, taskHash [sha1.Size]byte, key string, value transfer.JSONInterface) errors.Error {
	om.Lock()
	defer om.Unlock()

	taskHashKey := hex.EncodeToString(taskHash[:])

	if key == "" {
		return errors.NewOutputsKeyEmptyError()
	}

	if om.data == nil {
		om.data = make(map[string]map[string]transfer.JSONInterface)
	}

	if om.data[taskHashKey] == nil {
		om.data[taskHashKey] = make(map[string]transfer.JSONInterface)
	}

	om.data[taskHashKey][key] = value

	return nil
}

func New() *OutputsManager {
	return &OutputsManager{}
}
