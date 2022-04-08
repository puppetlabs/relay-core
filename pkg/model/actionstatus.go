package model

import (
	"context"
)

type ActionStatus struct {
	ExitCode int
}

type ActionStatusGetterManager interface {
	Get(ctx context.Context, action Action) (*ActionStatus, error)
}

type ActionStatusSetterManager interface {
	Set(ctx context.Context, ss *ActionStatus) error
}

type ActionStatusManager interface {
	ActionStatusGetterManager
	ActionStatusSetterManager
}
