package model

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/puppetlabs/leg/jsonutil/pkg/jsonpath"
)

type UnresolvableError struct {
	Causes []error
}

func (e *UnresolvableError) Error() string {
	var causes []string
	for _, err := range e.Causes {
		causes = append(causes, fmt.Sprintf("* %s", err.Error()))
	}

	return fmt.Sprintf("unresolvable:\n%s", strings.Join(causes, "\n"))
}

type DataNotFoundError struct {
	Name string
}

func (e *DataNotFoundError) Error() string {
	if e.Name == "" {
		return "model: data could not be found"
	}

	return fmt.Sprintf("model: %s data could not be found", e.Name)
}

type SecretNotFoundError struct {
	Name string
}

func (e *SecretNotFoundError) Error() string {
	return fmt.Sprintf("model: secret %q could not be found", e.Name)
}

type ConnectionNotFoundError struct {
	Type string
	Name string
}

func (e *ConnectionNotFoundError) Error() string {
	return fmt.Sprintf("model: connection type %q with name %q could not be found", e.Type, e.Name)
}

type OutputNotFoundError struct {
	From string
	Name string
}

func (e *OutputNotFoundError) Error() string {
	return fmt.Sprintf("model: output %q of step %q could not be found", e.Name, e.From)
}

type ParameterNotFoundError struct {
	Name string
}

func (e *ParameterNotFoundError) Error() string {
	return fmt.Sprintf("model: parameter %q could not be found", e.Name)
}

type AnswerNotFoundError struct {
	AskRef string
	Name   string
}

func (e *AnswerNotFoundError) Error() string {
	return fmt.Sprintf("model: answer %q of ask %q could not be found", e.Name, e.AskRef)
}

type FunctionResolutionError struct {
	Name  string
	Cause error
}

func (e *FunctionResolutionError) Unwrap() error {
	return e.Cause
}

func (e *FunctionResolutionError) Error() string {
	return fmt.Sprintf("model: function %q could not be invoked: %+v", e.Name, e.Cause)
}

type UnsupportedValueError struct {
	Type reflect.Type
}

var _ jsonpath.PropagatableError = &UnsupportedValueError{}

func (e *UnsupportedValueError) Error() string {
	return fmt.Sprintf("could not evaluate a value of type %s, must be a YAML-compatible type", e.Type)
}

func (e *UnsupportedValueError) Propagate() bool { return true }

type InvocationError struct {
	Name  string
	Cause error
}

var _ jsonpath.PropagatableError = &InvocationError{}

func (e *InvocationError) Unwrap() error {
	return e.Cause
}

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

func (e *PathEvaluationError) Unwrap() error {
	return e.Cause
}

func (e *PathEvaluationError) Error() string {
	path, err := e.trace()
	return fmt.Sprintf("path %q: %+v", strings.Join(path, "."), err)
}

func (e *PathEvaluationError) Propagate() bool { return true }
