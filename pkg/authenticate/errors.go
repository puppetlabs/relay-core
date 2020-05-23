package authenticate

import (
	"errors"
	"fmt"
)

var (
	ErrMalformed = errors.New("authenticate: malformed token")
)

type NotFoundError struct {
	Reason string
	Causes []error
}

func (e *NotFoundError) Error() string {
	message := "authenticate: "
	if e.Reason == "" {
		message += "not found"
	} else {
		message += e.Reason
	}

	switch len(e.Causes) {
	case 0:
	case 1:
		message += fmt.Sprintf(": %+v", e.Causes[0])
	default:
		message += ":"
		for _, cause := range e.Causes {
			message += fmt.Sprintf("\n* %+v", cause)
		}
	}

	return message
}
