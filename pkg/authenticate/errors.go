package authenticate

import "errors"

var (
	ErrMalformed = errors.New("authenticate: malformed token")
	ErrNotFound  = errors.New("authenticate: not found")
)
