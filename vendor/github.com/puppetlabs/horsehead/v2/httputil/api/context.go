package api

import (
	"context"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
)

type contextKey int

const (
	errorSensitivityContextKey contextKey = iota
)

func NewContextWithErrorSensitivity(ctx context.Context, sensitivity errawr.ErrorSensitivity) context.Context {
	return context.WithValue(ctx, errorSensitivityContextKey, sensitivity)
}

func ErrorSensitivityFromContext(ctx context.Context) (s errawr.ErrorSensitivity, ok bool) {
	s, ok = ctx.Value(errorSensitivityContextKey).(errawr.ErrorSensitivity)
	return
}
