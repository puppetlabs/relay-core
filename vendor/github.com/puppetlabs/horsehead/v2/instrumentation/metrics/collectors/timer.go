package collectors

type TimerHandle struct{}

type TimerOptions struct {
	Description         string
	Labels              []string
	HistogramBoundaries []float64
}

type Timer interface {
	WithLabels([]Label) (Timer, error)
	Start() *TimerHandle
	ObserveDuration(*TimerHandle)
}
