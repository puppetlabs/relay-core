package reject

import (
	"context"
	"time"

	"github.com/puppetlabs/relay-core/pkg/model"
)

type timerManager struct{}

func (*timerManager) Get(ctx context.Context, name string) (*model.Timer, error) {
	return nil, model.ErrRejected
}

func (*timerManager) Set(ctx context.Context, name string, t time.Time) (*model.Timer, error) {
	return nil, model.ErrRejected
}

var TimerManager model.TimerManager = &timerManager{}
