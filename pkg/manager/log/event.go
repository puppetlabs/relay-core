package log

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type eventManager struct{}

func (*eventManager) Emit(ctx context.Context, data map[string]interface{}) (*model.Event, error) {
	log(ctx).Info("received event", "data", data)

	return &model.Event{
		Data: data,
	}, nil
}

var EventManager model.EventManager = &eventManager{}
