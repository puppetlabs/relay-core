package noop

import "github.com/puppetlabs/horsehead/v2/instrumentation/metrics/collectors"

type Counter struct{}

func (c Counter) WithLabels([]collectors.Label) (collectors.Counter, error) { return c, nil }
func (c Counter) Inc()                                                      {}
func (c Counter) Add(float64)                                               {}
