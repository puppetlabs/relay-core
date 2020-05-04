package authenticate

import "context"

const (
	ControllerIssuer = "controller.k8s.relay.sh"

	MetadataAPIAudienceV1 = "k8s.relay.sh/metadata-api/v1"
)

type Issuer interface {
	Issue(ctx context.Context, claims *Claims) (Raw, error)
}

type IssuerFunc func(ctx context.Context, claims *Claims) (Raw, error)

var _ Issuer = IssuerFunc(nil)

func (isf IssuerFunc) Issue(ctx context.Context, claims *Claims) (Raw, error) {
	return isf(ctx, claims)
}
