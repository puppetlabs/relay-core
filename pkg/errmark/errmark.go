package errmark

type Marker interface {
	Map(fn func(err error) error) error
}

type MarkedError struct {
	Delegate error
	Maps     []func(err error) error
}

func (e *MarkedError) Error() string {
	return e.Resolve().Error()
}

func (e *MarkedError) MapFirst(fn func(err error) error) *MarkedError {
	e.Maps = append([]func(err error) error{fn}, e.Maps...)
	return e
}

func (e *MarkedError) MapLast(fn func(err error) error) *MarkedError {
	e.Maps = append(e.Maps, fn)
	return e
}

func (e *MarkedError) Resolve() error {
	err := e.Delegate

	for _, m := range e.Maps {
		if marker, ok := err.(Marker); ok {
			err = marker.Map(m)
		} else {
			err = m(err)
		}
	}

	return err
}

func AsMarkedError(err error) *MarkedError {
	marker, ok := err.(*MarkedError)
	if !ok {
		marker = &MarkedError{Delegate: err}
	}

	return marker
}

func MapFirst(err error, fn func(err error) error) error {
	return AsMarkedError(err).MapFirst(fn)
}

func MapLast(err error, fn func(err error) error) error {
	return AsMarkedError(err).MapLast(fn)
}

func Resolve(err error) error {
	return AsMarkedError(err).Resolve()
}

type IfFunc func(err error, fn func(err error))

func IfAll(err error, conds []IfFunc, fn func(err error)) {
	for i := len(conds) - 1; i >= 0; i-- {
		fn = func(cond IfFunc, next func(err error)) func(err error) {
			return func(err error) { cond(err, next) }
		}(conds[i], fn)
	}

	fn(Resolve(err))
}

func Is(candidate, wanted error) bool {
	return AsMarkedError(candidate).Delegate == wanted
}
