package alertstest

import (
	"context"

	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
)

type ReporterRecorder struct {
	Err  error
	Tags []trackers.Tag
}

func (r ReporterRecorder) WithNewTrace() trackers.Reporter {
	return r
}

func (r ReporterRecorder) WithTrace(t *trackers.Trace) trackers.Reporter {
	return r
}

func (r ReporterRecorder) WithTags(tags ...trackers.Tag) trackers.Reporter {
	return &ReporterRecorder{
		Tags: append(append([]trackers.Tag{}, r.Tags...), tags...),
	}
}

func (r ReporterRecorder) AsWarning() trackers.Reporter {
	return r
}

func (r ReporterRecorder) Report(ctx context.Context) <-chan error {
	ch := make(chan error, 1)
	ch <- nil
	return ch
}

func (r ReporterRecorder) ReportSync(ctx context.Context) error {
	return <-r.Report(ctx)
}
