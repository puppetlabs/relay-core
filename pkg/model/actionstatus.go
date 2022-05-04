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
		if as.ProcessState != nil && as.ProcessState.ExitCode != 0 {
			return true, nil
		}
	case StatusPropertySkipped:
		if as.WhenCondition != nil {
			switch as.WhenCondition.WhenConditionStatus {
			case WhenConditionStatusFailure, WhenConditionStatusNotSatisfied:
				return true, nil
			case WhenConditionStatusSatisfied:
				return false, nil
			}
		}
	case StatusPropertySucceeded:
		if as.ProcessState != nil && as.ProcessState.ExitCode == 0 {
			return true, nil
		}
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
