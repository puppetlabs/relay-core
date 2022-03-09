package model

import (
	"context"
	"time"

	"github.com/puppetlabs/relay-core/pkg/expr/parse"
)

type ConditionEvaluationResult struct {
	Expression parse.Tree
}

type SchemaValidationResult struct {
	Expression parse.Tree
}

type StepMessage struct {
	ID                        string
	Details                   string
	Time                      time.Time
	ConditionEvaluationResult *ConditionEvaluationResult
	SchemaValidationResult    *SchemaValidationResult
}

type StepMessageGetterManager interface {
	List(ctx context.Context) ([]*StepMessage, error)
}

type StepMessageSetterManager interface {
	Set(ctx context.Context, sm *StepMessage) error
}

type StepMessageManager interface {
	StepMessageGetterManager
	StepMessageSetterManager
}
