package collectors

// CounterOptions is used to configure a counter metric with labels
// that must be used when incrementing.
type CounterOptions struct {
	Description string
	Labels      []string
}

// Counter is a metric that always counts up. You can either Add() a number
// of 0 or more, or use Inc() to increment the count by one.
type Counter interface {
	// WithLabels takes a slice of Labels and returns a new Counter with
	// those label attached.
	WithLabels([]Label) (Counter, error)
	// Inc increments the count for the configured metric in the backend.
	Inc()
	// Add takes a float64 that must be a positive number >= 0. If the float64
	// is < 0 then Add will panic.
	Add(float64)
}
