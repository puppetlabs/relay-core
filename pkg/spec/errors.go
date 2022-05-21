package spec

import (
	"errors"
	"fmt"
	"strings"

	"github.com/puppetlabs/leg/relspec/pkg/ref"
)

var (
	ErrNotFound = errors.New("not found")
)

type FieldNotFoundError struct {
	Name string
}

func (e *FieldNotFoundError) Error() string {
	return fmt.Sprintf("the required field %q could not be found", e.Name)
}

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

func AddUnresolvableErrorsTo[T ref.ID[T]](to *UnresolvableError, from *ref.Log[T]) {
	from.ForEach(func(rf ref.Reference[T]) {
		if err := rf.Error(); err != nil {
			to.Causes = append(to.Causes, fmt.Errorf("%v: %w", rf.ID(), err))
		}
	})
}
