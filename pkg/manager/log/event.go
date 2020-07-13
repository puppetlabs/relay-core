package log

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type eventManager struct{}

func (*eventManager) Emit(ctx context.Context, data map[string]interface{}, key string) (*model.Event, error) {
	log(ctx).Info("received event", "data", data, "key", key)

	return &model.Event{
		Data: data,
		Key:  key,
	}, nil
}

var EventManager model.EventManager = &eventManager{}
