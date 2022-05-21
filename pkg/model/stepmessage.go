package model

import (
	"context"
	"time"
)

type ConditionEvaluationResult struct {
	Expression any
}

type SchemaValidationResult struct {
	Expression any
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
