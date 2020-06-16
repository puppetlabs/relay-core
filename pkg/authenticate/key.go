package authenticate

import (
	"context"

	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type KeyResolver struct {
	key         interface{}
	expectation jwt.Expected
}

var _ Resolver = &KeyResolver{}

func (kr *KeyResolver) Resolve(ctx context.Context, state *Authentication, raw Raw) (*Claims, error) {
	tok, err := jwt.ParseSigned(string(raw))
	if err != nil {
		// This class of errors basically just means that the JWT itself is
		// malformed.
		return nil, &NotFoundError{Reason: "key: JWT parse error", Causes: []error{err}}
	}

	claims := &Claims{}
	if err := tok.Claims(kr.key, claims); err != nil {
		// And this class of error means that the token isn't valid per the key.
		return nil, &NotFoundError{Reason: "key: could not validate JWT signature", Causes: []error{err}}
	}

	if err := claims.Validate(kr.expectation); err != nil {
		return nil, &NotFoundError{Reason: "key: could not validate JWT claims", Causes: []error{err}}
	}

	return claims, nil
}

type KeyResolverOption func(kr *KeyResolver)

func KeyResolverWithExpectation(e jwt.Expected) KeyResolverOption {
	return func(kr *KeyResolver) {
		kr.expectation = e
	}
}

func NewKeyResolver(key interface{}, opts ...KeyResolverOption) *KeyResolver {
	kr := &KeyResolver{
		key: key,
	}

	for _, opt := range opts {
		opt(kr)
	}

	return kr
}

type KeySignerIssuer struct {
	signer jose.Signer
}

var _ Issuer = &KeySignerIssuer{}

func (ksi *KeySignerIssuer) Issue(ctx context.Context, claims *Claims) (Raw, error) {
	tok, err := jwt.Signed(ksi.signer).Claims(claims).CompactSerialize()
	if err != nil {
		return nil, err
	}

	return Raw(tok), nil
}

func NewKeySignerIssuer(signer jose.Signer) *KeySignerIssuer {
	return &KeySignerIssuer{
		signer: signer,
	}
}

func NewHS256KeySignerIssuer(key []byte) (*KeySignerIssuer, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.HS256, Key: key}, &jose.SignerOptions{})
	if err != nil {
		return nil, err
	}

	return NewKeySignerIssuer(signer), nil
}
