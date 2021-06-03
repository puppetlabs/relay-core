package model

import "context"

type contextKey int

const (
	unresolvableContextKey contextKey = iota
)

func ContextWithUnresolvable(ctx context.Context, u *Unresolvable) context.Context {
	return context.WithValue(ctx, unresolvableContextKey, u)
}

func ContextWithNewUnresolvable(ctx context.Context) (context.Context, *Unresolvable) {
	u := &Unresolvable{}
	return ContextWithUnresolvable(ctx, u), u
}

func UnresolvableFromContext(ctx context.Context) *Unresolvable {
	u, ok := ctx.Value(unresolvableContextKey).(*Unresolvable)
	if !ok {
		u = &Unresolvable{}
	}

	return u
}
