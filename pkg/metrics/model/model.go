package model

const (
	MetricEventTypeNormal    = "event_type_normal"
	MetricEventTypeWarning   = "event_type_warning"
	MetricWorkflowRunCount   = "workflow_run_count"
	MetricWorkflowRunOutcome = "workflow_run_outcome"
	MetricWorkflowRunStatus  = "workflow_run_status"

	// MetricWorkflowRunExecutionTimeSeconds is the amount of time the workflow
	// run spent being processed after a controller picked it up (difference
	// between status end time and start time).
	MetricWorkflowRunExecutionTimeSeconds = "workflow_run_execution_time_seconds"

	// MetricWorkflowRunInitTimeSeconds is the amount of time spent between
	// creating the workflow and the first step's code actually running as
	// determined by the timing data reported by the entrypoint.
	MetricWorkflowRunInitTimeSeconds = "workflow_run_init_time_seconds"

	// MetricWorkflowRunTotalTime is the total amount of time the workflow spent
	// in the cluster before completing.
	MetricWorkflowRunTotalTimeSeconds = "workflow_run_total_time_seconds"

	MetricAttributeReason  = "reason"
	MetricAttributeOutcome = "outcome"
	MetricAttributeStatus  = "status"
)

type EventFilter struct {
	Metric  string
	Filters []string
}
