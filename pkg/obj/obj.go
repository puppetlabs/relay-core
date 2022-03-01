package obj

import "context"

type Configurable interface {
	Configure(ctx context.Context) error
}
