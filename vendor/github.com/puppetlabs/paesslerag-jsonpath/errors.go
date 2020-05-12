package jsonpath

import "fmt"

// ForcePropagation propagates an error even when a selector would otherwise
// drop it, indicating, e.g., problems with the underlying data.
type ForcePropagation struct {
	Cause error
}

func (e *ForcePropagation) Error() string {
	return fmt.Sprintf("jsonpath: force propagation: %+v", e.Cause)
}
