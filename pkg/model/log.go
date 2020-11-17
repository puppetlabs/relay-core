package model

import (
	"context"
)

type LogManager interface {
	PostLog(ctx context.Context, value interface{}) ([]byte, error)
	PostLogMessage(ctx context.Context, logID string, value interface{}) ([]byte, error)
}
