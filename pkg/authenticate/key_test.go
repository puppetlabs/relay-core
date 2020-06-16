package authenticate_test

import (
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"testing"

	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	jose "gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

func TestKeyResolver(t *testing.T) {
	ctx := context.Background()

	pub := &rsa.PublicKey{}
	pub.N, _ = (&big.Int{}).SetString("23229357202741919638383280138522633022853944843431010660695240731243983781955822044076260906998420081070495758657355484080902688205488917574722610214038346603877742848183108168372601718810240667773893759953250384250771035412963193165206663559256721354711940912606303263122490374357575355776291664758127350120185973674946540008547109293863899079166010121223800013599343358241764582017568808789269290939419433419338707872087305260975469901788488345432212226379383965733016549323138990524646376657991265704069850641506890139170757406478357701818954293478134814138413187490242820937086502201803214959090864954196914312313", 10)
	pub.E = 65537

	tok := authenticate.Raw("eyJhbGciOiJSUzUxMiIsImtpZCI6IiJ9.eyJzdWIiOiJmb28ifQ.jQwtxNXlvOmwURjPA59EWtmiC6QOBEulyX5S7E-IpRrm9FNCsW0uU5KOnebjliI8kGVYjgaOr7bohA6dn7MjRm_L7Gwch5ZP42VUbzNy12eLhzan7MFfeM7vj4PqVAJcZFqnQzLZsuQkAiq-o-Y-3V21QY90nSiLk_wlq4h7ibGhfHobi50RRnP33vHhkkq0E1W7Setz-vrVDwTf93O5S4qk7i8VwhNEsD-Dt8cF-mYLQpdgfjLHmEwyXPD29ZcyVRd3wgkIrkrWgjdauLxxbDlo8l6h41GbCY323_ieBH5P1b1vlu5pBu_dFDiq2AyYZ_Itt7DrlzF17zyyRCzY7w")

	claims, err := authenticate.NewKeyResolver(pub, authenticate.KeyResolverWithExpectation(jwt.Expected{Subject: "foo"})).Resolve(ctx, authenticate.NewAuthentication(), tok)
	require.NoError(t, err)
	require.Equal(t, "foo", claims.Subject)
}

func TestKeySignerIssuer(t *testing.T) {
	ctx := context.Background()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: key}, &jose.SignerOptions{})
	require.NoError(t, err)

	raw, err := authenticate.NewKeySignerIssuer(signer).Issue(ctx, &authenticate.Claims{Claims: &jwt.Claims{Subject: "foo"}})
	require.NoError(t, err)

	tok, err := jwt.ParseSigned(string(raw))
	require.Len(t, tok.Headers, 1)
	require.Equal(t, string(jose.RS512), tok.Headers[0].Algorithm)

	claims := &authenticate.Claims{}
	require.NoError(t, tok.Claims(key.Public(), claims))
	require.Equal(t, "foo", claims.Subject)
}

func TestKeySignerIssuerThenResolver(t *testing.T) {
	ctx := context.Background()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS512, Key: key}, &jose.SignerOptions{})
	require.NoError(t, err)

	raw, err := authenticate.NewKeySignerIssuer(signer).Issue(ctx, &authenticate.Claims{Claims: &jwt.Claims{Subject: "foo"}})
	require.NoError(t, err)

	claims, err := authenticate.NewKeyResolver(key.Public()).Resolve(ctx, authenticate.NewAuthentication(), raw)
	require.NoError(t, err)
	require.Equal(t, "foo", claims.Subject)
}
