package model

import (
	"context"
	"time"
)

const (
	TimerStepInit = "relay.step.init"
)

type Timer struct {
	Name string
	Time time.Time
}

type TimerGetterManager interface {
	Get(ctx context.Context, name string) (*Timer, error)
}

type TimerSetterManager interface {
	Set(ctx context.Context, name string, t time.Time) (*Timer, error)
}

type TimerManager interface {
	TimerGetterManager
	TimerSetterManager
}
