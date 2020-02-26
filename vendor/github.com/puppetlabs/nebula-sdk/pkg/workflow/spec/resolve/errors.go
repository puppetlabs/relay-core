package resolve

import "fmt"

type SecretNotFoundError struct {
	Name string
}

func (e *SecretNotFoundError) Error() string {
	return fmt.Sprintf("resolve: secret %q could not be found", e.Name)
}

type OutputNotFoundError struct {
	From string
	Name string
}

func (e *OutputNotFoundError) Error() string {
	return fmt.Sprintf("resolve: output %q of step %q could not be found", e.Name, e.From)
}

type ParameterNotFoundError struct {
	Name string
}

func (e *ParameterNotFoundError) Error() string {
	return fmt.Sprintf("resolve: parameter %q could not be found", e.Name)
}

type AnswerNotFoundError struct {
	AskRef string
	Name   string
}

func (e *AnswerNotFoundError) Error() string {
	return fmt.Sprintf("resolve: answer %q of ask %q could not be found", e.Name, e.AskRef)
}

type FunctionResolutionError struct {
	Name  string
	Cause error
}

func (e *FunctionResolutionError) Error() string {
	return fmt.Sprintf("resolve: function %q could not be invoked: %+v", e.Name, e.Cause)
}
