package memory

import (
	"bytes"
	"context"
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

	if om.data == nil {
		return nil, errors.NewOutputsTaskNotFound(taskName)
	}

	if om.data[taskName] == nil {
		return nil, errors.NewOutputsTaskNotFound(taskName)
	}

	if om.data[taskName][key] == nil {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	return &outputs.Output{
		TaskName: taskName,
		Key:      key,
		Value:    string(om.data[taskName][key]),
	}, nil
}

func (om *OutputsManager) Put(ctx context.Context, taskName, key string, value io.Reader) errors.Error {
	om.Lock()
	defer om.Unlock()

	if key == "" {
		return errors.NewOutputsKeyEmptyError()
	}

	if om.data == nil {
		om.data = make(map[string]map[string][]byte)
	}

	if om.data[taskName] == nil {
		om.data[taskName] = make(map[string][]byte)
	}

	buf := &bytes.Buffer{}
	_, err := buf.ReadFrom(value)
	if err != nil {
		return errors.NewOutputsValueReadError().WithCause(err)
	}

	om.data[taskName][key] = buf.Bytes()

	return nil
}

func New() *OutputsManager {
	return &OutputsManager{}
}
