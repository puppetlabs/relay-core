package collectors

// TimerHandle is a type used to map running timers for defered observations.
type TimerHandle struct{}

// TimerOptions is used to configure a timer with labels and histogram boundaries.
type TimerOptions struct {
	Description         string
	Labels              []string
	HistogramBoundaries []float64
}

// Timer is a named metric with labels. It allows concurrent use by returning a handle
// when the timer is started. This handle is used to lookup a running timer to record the duration.
type Timer interface {
	// WithLabels returns a new Timer with labels attached.
	WithLabels(...Label) Timer
	// Start starts the timer clock and returns a TimerHandle that represents a specific
	// clock that's ticking.
	Start() *TimerHandle
	// ObserveDuration takes a TimerHandle and stops the clock, recording the duration
	// that elapsed to the metrics backend. Optionally takes labels to apply. This is
	// useful if you want to record the duration on a metric labeled as a failure.
	ObserveDuration(*TimerHandle, ...Label)
}
