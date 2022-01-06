package run

import (
	"github.com/puppetlabs/leg/instrumentation/metrics"
	"github.com/puppetlabs/leg/instrumentation/metrics/collectors"
)

const (
	metricWorkflowRunStartUpDuration   = "workflow_run_startup_duration"
	metricWorkflowRunLogUploadDuration = "workflow_run_log_upload_duration"
)

type trackDurationOptions struct {
	accountID string
}

type trackDurationOption interface {
	apply(*trackDurationOptions)
}

type trackDurationOptionFunc struct {
	fn func(*trackDurationOptions)
}

func (o trackDurationOptionFunc) apply(opts *trackDurationOptions) {
	o.fn(opts)
}

func withAccountIDTrackDurationOption(accountID string) trackDurationOptionFunc {
	return trackDurationOptionFunc{
		fn: func(opts *trackDurationOptions) {
			opts.accountID = accountID
		},
	}
}

type controllerObservations struct {
	mets *metrics.Metrics
}

func (c *controllerObservations) trackDurationWithOutcome(metric string, fn func() error, opts ...trackDurationOption) error {
	o := trackDurationOptions{
		accountID: "none",
	}

	for _, opt := range opts {
		opt.apply(&o)
	}

	timer := c.mets.MustTimer(metric)
	handle := timer.Start()
	outcomeLabel := collectors.Label{Name: "outcome", Value: "success"}

	err := fn()
	if err != nil {
		outcomeLabel.Value = "failed"
	}

	accountIDLabel := collectors.Label{Name: "account_id", Value: o.accountID}

	timer.ObserveDuration(handle, outcomeLabel, accountIDLabel)

	return err
}

func newControllerObservations(mets *metrics.Metrics) *controllerObservations {
	mets.MustRegisterTimer(metricWorkflowRunStartUpDuration, collectors.TimerOptions{
		Description: "duration of fully starting a workflow run",
		Labels:      []string{"outcome", "account_id"},
	})

	mets.MustRegisterTimer(metricWorkflowRunLogUploadDuration, collectors.TimerOptions{
		Description: "time spent waiting for the step logs to upload",
		Labels:      []string{"outcome", "account_id"},
	})

	return &controllerObservations{
		mets: mets,
	}
}
