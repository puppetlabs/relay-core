package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/cert"
)

var caConfig = cert.Config{
	CommonName:   "Puppet Relay Integration Test CA",
	Organization: []string{"Puppet"},
}

type CertificateBundle struct {
	AuthorityPEM         []byte
	BundlePEM            []byte
	ServerCertificatePEM []byte
	ServerKeyPEM         []byte
}

func GenerateCertificateBundle(t *testing.T, cn string) *CertificateBundle {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	ca, err := cert.NewSelfSignedCACert(caConfig, caKey)
	require.NoError(t, err)

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	serverKeyRaw, err := x509.MarshalPKCS8PrivateKey(serverKey)
	require.NoError(t, err)

	now := time.Now().UTC()
	serverCertTemplate := &x509.Certificate{
		Subject:      pkix.Name{CommonName: cn, Organization: []string{"Puppet"}},
		DNSNames:     []string{cn},
		SerialNumber: big.NewInt(1),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		NotBefore:    now,
		NotAfter:     now.Add(24 * time.Hour),
	}

	serverCertRaw, err := x509.CreateCertificate(rand.Reader, serverCertTemplate, ca, serverKey.Public(), caKey)
	require.NoError(t, err)

	cb := &CertificateBundle{
		AuthorityPEM:         pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.Raw}),
		ServerCertificatePEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverCertRaw}),
		ServerKeyPEM:         pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: serverKeyRaw}),
	}
	cb.BundlePEM = append(append([]byte{}, cb.ServerCertificatePEM...), cb.AuthorityPEM...)
	return cb
}
