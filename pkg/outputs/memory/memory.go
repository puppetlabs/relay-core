package memory

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"sync"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
)

type OutputsManager struct {
	data map[string]map[string][]byte

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

	if om.data[taskHashKey][key] == nil {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	return &outputs.Output{
		TaskName: taskName,
		Key:      key,
		Value:    string(om.data[taskHashKey][key]),
	}, nil
}

func (om *OutputsManager) Put(ctx context.Context, taskHash [sha1.Size]byte, key string, value io.Reader) errors.Error {
	om.Lock()
	defer om.Unlock()

	taskHashKey := hex.EncodeToString(taskHash[:])

	if key == "" {
		return errors.NewOutputsKeyEmptyError()
	}

	if om.data == nil {
		om.data = make(map[string]map[string][]byte)
	}

	if om.data[taskHashKey] == nil {
		om.data[taskHashKey] = make(map[string][]byte)
	}

	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(value)
	if err != nil {
		return errors.NewOutputsValueReadError().WithCause(err)
	}

	om.data[taskHashKey][key] = buf.Bytes()

	return nil
}

func New() *OutputsManager {
	return &OutputsManager{}
}
