package errmark

type UserError struct {
	Delegate error
}

var _ Marker = &UserError{}

func (te *UserError) Error() string {
	return te.Delegate.Error()
}

func (te *UserError) Map(fn func(err error) error) error {
	te.Delegate = fn(te.Delegate)
	return te
}

func MarkUser(err error) error {
	return MapFirst(err, func(err error) error {
		return &UserError{Delegate: err}
	})
}

func IfUser(err error, fn func(err error)) {
	if te, ok := err.(*UserError); ok {
		fn(te.Delegate)
	}
}

func IfNotUser(err error, fn func(err error)) {
	if _, ok := err.(*UserError); !ok {
		fn(err)
	}
}
