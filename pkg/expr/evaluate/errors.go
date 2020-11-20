package evaluate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	jsonpath "github.com/puppetlabs/paesslerag-jsonpath"
)

var (
	ErrUnsupportedLanguage = errors.New("evaluate: unsupported language")
)

type UnsupportedValueError struct {
	Type reflect.Type
}

var _ jsonpath.PropagatableError = &UnsupportedValueError{}

func (e *UnsupportedValueError) Error() string {
	return fmt.Sprintf("could not evaluate a value of type %s, must be a JSON-compatible type", e.Type)
}

func (e *UnsupportedValueError) Propagate() bool { return true }

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

type InvalidEncodingError struct {
	Type  string
	Cause error
}

var _ jsonpath.PropagatableError = &InvalidEncodingError{}

func (e *InvalidEncodingError) Error() string {
	return fmt.Sprintf("could not evaluate encoding %q: %+v", e.Type, e.Cause)
}

func (e *InvalidEncodingError) Propagate() bool { return true }

type InvalidInvocationError struct {
	Name  string
	Cause error
}

var _ jsonpath.PropagatableError = &InvalidInvocationError{}

func (e *InvalidInvocationError) Error() string {
	return fmt.Sprintf("could not evaluate function invocation %q: %+v", e.Name, e.Cause)
}

func (e *InvalidInvocationError) Propagate() bool { return true }

type InvocationError struct {
	Name  string
	Cause error
}

var _ jsonpath.PropagatableError = &InvocationError{}

func (e *InvocationError) Error() string {
	return fmt.Sprintf("invocation of function %q failed: %+v", e.Name, e.Cause)
}

func (e *InvocationError) Propagate() bool { return true }

type PathEvaluationError struct {
	Path  string
	Cause error
}

var _ jsonpath.PropagatableError = &PathEvaluationError{}

func (e *PathEvaluationError) trace() ([]string, error) {
	var path []string
	for {
		path = append(path, e.Path)

		en, ok := e.Cause.(*PathEvaluationError)
		if !ok {
			return path, e.Cause
		}

		e = en
	}
}

func (e *PathEvaluationError) UnderlyingCause() error {
	_, err := e.trace()
	return err
}

func (e *PathEvaluationError) Error() string {
	path, err := e.trace()
	return fmt.Sprintf("path %q: %+v", strings.Join(path, "."), err)
}

func (e *PathEvaluationError) Propagate() bool { return true }
