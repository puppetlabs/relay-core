package evaluate

import (
	"fmt"

	"github.com/puppetlabs/leg/jsonutil/pkg/jsonpath"
)

type InvalidTypeError struct {
	Type  string
	Cause error
}

var _ jsonpath.PropagatableError = &InvalidTypeError{}

func (e *InvalidTypeError) Error() string {
	return fmt.Sprintf("could not evaluate a %s type: %+v", e.Type, e.Cause)
}

func (e *InvalidTypeError) Propagate() bool { return true }

type FieldNotFoundError struct {
	Name string
}

func (e *FieldNotFoundError) Error() string {
	return fmt.Sprintf("the required field %q could not be found", e.Name)
}

type DataResolverNotFoundError struct {
	Name string
}

func (e *DataResolverNotFoundError) Error() string {
	if e.Name == "" {
		return "default data resolver could not be found"
	}

	return fmt.Sprintf("data resolver %q could not be found", e.Name)
}

type DataQueryError struct {
	Query string
	Cause error
}

func (e *DataQueryError) Unwrap() error {
	return e.Cause
}

func (e *DataQueryError) Error() string {
	return fmt.Sprintf("query %q: %v", e.Query, e.Cause)
}

type InvalidEncodingError struct {
	Type  string
	Cause error
}

var _ jsonpath.PropagatableError = &InvalidEncodingError{}

func (e *InvalidEncodingError) Error() string {
	return fmt.Sprintf("could not evaluate encoding %q: %+v", e.Type, e.Cause)
}

func (e *InvalidEncodingError) Propagate() bool { return true }
