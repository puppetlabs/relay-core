package obj

import (
	"context"
	"reflect"
)

type Configurable interface {
	Configure(ctx context.Context) error
}

type IgnoreNilConfigurable struct {
	Configurable
}

func (inc IgnoreNilConfigurable) Configure(ctx context.Context) error {
	if inc.Configurable == nil || reflect.ValueOf(inc.Configurable).IsNil() {
		return nil
	}

	return inc.Configurable.Configure(ctx)
}
