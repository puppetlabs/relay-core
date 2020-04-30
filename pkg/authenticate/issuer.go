package authenticate

import "context"

type Issuer interface {
	Issue(ctx context.Context, claims *Claims) (Raw, error)
}
