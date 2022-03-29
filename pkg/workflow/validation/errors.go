package validation

import "fmt"

type SchemaDoesNotExistError struct {
	Name string
}

func (e *SchemaDoesNotExistError) Error() string {
	return fmt.Sprintf("schema for %s does not exist in the schema registry", e.Name)
}

type StepMetadataFetchError struct {
	StatusCode int
}

func (e *StepMetadataFetchError) Error() string {
	return fmt.Sprintf("step metadata entrypoint responded with %d status code", e.StatusCode)
}
