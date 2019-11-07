package collectors

type CounterOptions struct {
	Description string
	Labels      []string
}

type Counter interface {
	WithLabels([]Label) (Counter, error)
	Inc()
	Add(float64)
}
