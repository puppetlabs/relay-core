package workflow

import (
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics"
	"github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"
)

const (
	metricWorkflowRunStartUpDuration                   = "workflow_run_startup_duration"
	metricWorkflowRunWaitForMetadataAPIServiceDuration = "workflow_run_wait_for_metadata_api_service_duration"
	metricWorkflowRunLogUploadDuration                 = "workflow_run_log_upload_duration"
)

type controllerObservations struct {
	mets *metrics.Metrics
}

func (c *controllerObservations) trackDurationWithOutcome(metric string, fn func() error) error {
	timer := c.mets.MustTimer(metric)
	handle := timer.Start()
	label := collectors.Label{Name: "outcome", Value: "success"}

	err := fn()
	if err != nil {
		label.Value = "failed"
	}

	timer.ObserveDuration(handle, label)

	return err
}

func newControllerObservations(mets *metrics.Metrics) *controllerObservations {
	mets.MustRegisterTimer(metricWorkflowRunStartUpDuration, collectors.TimerOptions{
		Description: "duration of fully starting a workflow run",
		Labels:      []string{"outcome"},
	})

	mets.MustRegisterTimer(metricWorkflowRunWaitForMetadataAPIServiceDuration, collectors.TimerOptions{
		Description: "time spent waiting for the metadata api service to be available",
		Labels:      []string{"outcome"},
	})

	mets.MustRegisterTimer(metricWorkflowRunLogUploadDuration, collectors.TimerOptions{
		Description: "time spent waiting for the step logs to upload",
		Labels:      []string{"outcome"},
	})

	return &controllerObservations{
		mets: mets,
	}
}
