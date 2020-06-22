package errmark

type Marker interface {
	Map(fn func(err error) error) Marker
	Resolve() error
}

type Error struct {
	Delegate error
}

var _ Marker = &Error{}

func (e *Error) Map(fn func(err error) error) Marker {
	e.Delegate = fn(e.Delegate)
	return e
}

func (e *Error) Resolve() error {
	return e.Delegate
}
