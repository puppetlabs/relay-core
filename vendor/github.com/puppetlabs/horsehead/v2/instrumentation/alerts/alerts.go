package alerts

import (
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/internal/noop"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/internal/sentry"
	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
	"github.com/puppetlabs/horsehead/v2/instrumentation/errors"
)

type Options struct {
	Environment string
	Version     string
}

type DelegateFunc func(opts Options) Delegate

func NoDelegate(opts Options) Delegate {
	return &noop.NoOp{}
}

func DelegateToSentry(dsn string) (DelegateFunc, errors.Error) {
	b, err := sentry.NewBuilder(dsn)
	if err != nil {
		return nil, err
	}

	fn := func(opts Options) Delegate {
		return b.WithEnvironment(opts.Environment).
			WithRelease(opts.Version).
			Build()
	}
	return fn, nil
}

type Alerts struct {
	delegate Delegate
}

func (a *Alerts) NewCapturer() trackers.Capturer {
	return a.delegate.NewCapturer()
}

func NewAlerts(fn DelegateFunc, opts Options) *Alerts {
	return &Alerts{
		delegate: fn(opts),
	}
}
