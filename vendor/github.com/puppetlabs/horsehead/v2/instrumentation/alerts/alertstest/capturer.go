package alertstest

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/puppetlabs/horsehead/v2/instrumentation/alerts/trackers"
)

type Capturer struct {
	Tags []trackers.Tag

	ReporterRecorders []*ReporterRecorder
}

func (c *Capturer) WithNewTrace() trackers.Capturer {
	return c
}

func (c *Capturer) WithAppPackages(packages []string) trackers.Capturer {
	return c
}

func (c *Capturer) WithUser(u trackers.User) trackers.Capturer {
	return c
}

func (c *Capturer) WithTags(tags ...trackers.Tag) trackers.Capturer {
	c.Tags = append(append([]trackers.Tag{}, c.Tags...), tags...)

	return c
}

func (c *Capturer) Try(ctx context.Context, fn func(ctx context.Context)) (rv interface{}) {
	defer func() {
		rv = recover()
		if nil != rv {
			debug.PrintStack()
		}
	}()

	fn(ctx)
	return nil
}

func (c *Capturer) Capture(err error) trackers.Reporter {
	rr := &ReporterRecorder{
		Err:  err,
		Tags: c.Tags,
	}

	c.ReporterRecorders = append(c.ReporterRecorders, rr)

	return rr
}

func (c *Capturer) CaptureMessage(message string) trackers.Reporter {
	return c.Capture(fmt.Errorf(message))
}

func (c *Capturer) Middleware() trackers.Middleware {
	return &Middleware{c: c}
}

func NewCapturer() *Capturer {
	return &Capturer{}
}
