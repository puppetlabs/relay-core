package model

const (
	MetricEventTypeNormal    = "event_type_normal"
	MetricEventTypeWarning   = "event_type_warning"
	MetricWorkflowRunCount   = "workflow_run_count"
	MetricWorkflowRunOutcome = "workflow_run_outcome"
	MetricWorkflowRunStatus  = "workflow_run_status"

	MetricAttributeReason  = "reason"
	MetricAttributeOutcome = "outcome"
	MetricAttributeStatus  = "status"
)

type EventFilter struct {
	Metric  string
	Filters []string
}
