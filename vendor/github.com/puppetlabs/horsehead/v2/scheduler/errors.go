package scheduler

import (
	"fmt"

	"github.com/puppetlabs/errawr-go/v2/pkg/errawr"
	"github.com/puppetlabs/horsehead/v2/scheduler/errors"
)

func coerceError(r interface{}) (err errawr.Error) {
	switch rt := r.(type) {
	case errawr.Error:
		err = rt
	case error:
		err = errors.NewProcessPanicError().WithCause(rt)
	default:
		err = errors.NewProcessPanicError().WithCause(fmt.Errorf("panic: %+v", rt))
	}

	return
}
