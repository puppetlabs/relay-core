package model

import (
	"context"
)

type StatusProperty string

const (
	StatusPropertyFailed    StatusProperty = "failed"
	StatusPropertySkipped   StatusProperty = "skipped"
	StatusPropertySucceeded StatusProperty = "succeeded"
)

func (sp StatusProperty) String() string {
	return string(sp)
}

type WhenConditionStatus string

const (
	WhenConditionStatusEvaluating   WhenConditionStatus = "WhenConditionEvaluating"
	WhenConditionStatusFailure      WhenConditionStatus = "WhenConditionFailure"
	WhenConditionStatusNotSatisfied WhenConditionStatus = "WhenConditionNotSatisfied"
	WhenConditionStatusSatisfied    WhenConditionStatus = "WhenConditionSatisfied"
	WhenConditionStatusUnknown      WhenConditionStatus = "WhenConditionUnknown"
)

func (wcs WhenConditionStatus) String() string {
	return string(wcs)
}

type ActionStatusProcessState struct {
	ExitCode int
}

type ActionStatusWhenCondition struct {
	WhenConditionStatus WhenConditionStatus
}

type ActionStatus struct {
	Name          string
	ProcessState  *ActionStatusProcessState
	WhenCondition *ActionStatusWhenCondition
}

func (as *ActionStatus) IsStatusProperty(property StatusProperty) (bool, error) {
	switch property {
	case StatusPropertyFailed:
		return as.Failed()
	case StatusPropertySkipped:
		return as.Skipped()
	case StatusPropertySucceeded:
		return as.Succeeded()
	default:
		return false, ErrNotFound
	}
}

func (as *ActionStatus) Failed() (bool, error) {
	if ok, err := as.Skipped(); ok && err == nil {
		return false, nil
	}

	if as.ProcessState != nil {
		if as.ProcessState.ExitCode != 0 {
			return true, nil
		}

		return false, nil
	}

	return false, ErrNotFound
}

func (as *ActionStatus) Skipped() (bool, error) {
	if as.WhenCondition != nil {
		switch as.WhenCondition.WhenConditionStatus {
		case WhenConditionStatusFailure, WhenConditionStatusNotSatisfied:
			return true, nil
		case WhenConditionStatusSatisfied:
			return false, nil
		}
	}

	return false, ErrNotFound
}

func (as *ActionStatus) Succeeded() (bool, error) {
	if skipped, err := as.Skipped(); skipped && err == nil {
		return false, nil
	}

	if failed, err := as.Failed(); err == nil {
		return !failed, nil
	}

	return false, ErrNotFound
}

type ActionStatusGetterManager interface {
	List(ctx context.Context) ([]*ActionStatus, error)
	Get(ctx context.Context, action Action) (*ActionStatus, error)
}

type ActionStatusSetterManager interface {
	Set(ctx context.Context, ss *ActionStatus) error
}

type ActionStatusManager interface {
	ActionStatusGetterManager
	ActionStatusSetterManager
}
