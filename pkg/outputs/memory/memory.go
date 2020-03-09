package memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/puppetlabs/horsehead/v2/encoding/transfer"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	"github.com/puppetlabs/nebula-tasks/pkg/outputs"
	"github.com/puppetlabs/nebula-tasks/pkg/task"
)

type OutputsManager struct {
	data map[string]map[string]transfer.JSONInterface

	sync.Mutex
}

func (om *OutputsManager) Get(ctx context.Context, metadata *task.Metadata, stepName, key string) (*outputs.Output, errors.Error) {
	om.Lock()
	defer om.Unlock()

	thisTask := &task.Task{
		Run:  metadata.Run,
		Name: stepName,
	}
	name := fmt.Sprintf("task-%s-outputs", thisTask.TaskHash().HexEncoding())

	if om.data == nil {
		return nil, errors.NewOutputsTaskNotFound(stepName)
	}

	if om.data[name] == nil {
		return nil, errors.NewOutputsTaskNotFound(stepName)
	}

	value, ok := om.data[name][key]
	if !ok {
		return nil, errors.NewOutputsKeyNotFound(key)
	}

	return &outputs.Output{
		TaskName: stepName,
		Key:      key,
		Value:    value,
	}, nil
}

func (om *OutputsManager) Put(ctx context.Context, metadata *task.Metadata, key string, value transfer.JSONInterface) errors.Error {
	om.Lock()
	defer om.Unlock()

	name := fmt.Sprintf("task-%s-outputs", metadata.Hash.HexEncoding())

	if key == "" {
		return errors.NewOutputsKeyEmptyError()
	}

	if om.data == nil {
		om.data = make(map[string]map[string]transfer.JSONInterface)
	}

	if om.data[name] == nil {
		om.data[name] = make(map[string]transfer.JSONInterface)
	}

	om.data[name][key] = value

	return nil
}

func New() *OutputsManager {
	return &OutputsManager{}
}
