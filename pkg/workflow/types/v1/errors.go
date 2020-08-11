package v1

import "fmt"

type WorkflowFileFormatError struct {
	Cause error
}

func (e *WorkflowFileFormatError) Unwrap() error {
	return e.Cause
}

func (e *WorkflowFileFormatError) Error() string {
	return "workflow file format error"
}

type WorkflowStepInvalidError struct {
	Name string
	Type string
}

func (e *WorkflowStepInvalidError) Error() string {
	return fmt.Sprintf("workflow step is invalid: %s %s", e.Name, e.Type)
}
