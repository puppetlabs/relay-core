package validation

import "fmt"

type SchemaDoesNotExistError struct {
	Name string
}

func (e *SchemaDoesNotExistError) Error() string {
	return fmt.Sprintf("schema for %s does not exist in the schema registry", e.Name)
}

type SchemaValidationError struct {
	Cause error
}

func (e *SchemaValidationError) Unwrap() error {
	return e.Cause
}

func (e *SchemaValidationError) Error() string {
	message := "data for the schema failed to validate"

	if e.Cause != nil {
		message += fmt.Sprintf("\n%s", e.Cause.Error())
	}

	return message
}

type StepMetadataFetchError struct {
	StatusCode int
}

func (e *StepMetadataFetchError) Error() string {
	return fmt.Sprintf("step metadata entrypoint responded with %d status code", e.StatusCode)
}
