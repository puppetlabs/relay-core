package authenticate_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/puppetlabs/nebula-tasks/pkg/authenticate"
	"github.com/puppetlabs/nebula-tasks/pkg/util/testutil"
	"github.com/stretchr/testify/require"
	"gopkg.in/square/go-jose.v2/jwt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestKubernetesIntermediary(t *testing.T) {
	ctx := context.Background()

	objs := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-test-namespace",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-test-namespace",
				Name:      "pod-a",
				Annotations: map[string]string{
					authenticate.KubernetesTokenAnnotation:   "my-auth-token",
					authenticate.KubernetesSubjectAnnotation: "my-test-subject",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.20.30.40",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-test-namespace",
				Name:      "pod-b",
			},
			Status: corev1.PodStatus{
				PodIP: "10.20.30.41",
			},
		},
	}

	kc := testutil.NewMockKubernetesClient(objs...)

	tests := []struct {
		IP            string
		ExpectedRaw   authenticate.Raw
		ExpectedError error
	}{
		{
			IP:          "10.20.30.40",
			ExpectedRaw: authenticate.Raw("my-auth-token"),
		},
		{
			IP:            "10.20.30.41",
			ExpectedError: authenticate.ErrNotFound,
		},
		{
			IP:            "10.20.30.42",
			ExpectedError: authenticate.ErrNotFound,
		},
	}
	for _, test := range tests {
		t.Run(test.IP, func(t *testing.T) {
			im := authenticate.NewKubernetesIntermediary(kc, net.ParseIP(test.IP))
			raw, err := im.Next(ctx, authenticate.NewAuthentication())
			require.Equal(t, test.ExpectedError, err)
			require.Equal(t, test.ExpectedRaw, raw)
		})
	}
}

func TestKubernetesIntermediaryChain(t *testing.T) {
	ctx := context.Background()

	objs := []runtime.Object{
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-test-namespace",
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "my-test-namespace",
				Name:      "pod-a",
				Annotations: map[string]string{
					authenticate.KubernetesTokenAnnotation:   "my-auth-token",
					authenticate.KubernetesSubjectAnnotation: "my-test-subject",
				},
			},
			Status: corev1.PodStatus{
				PodIP: "10.20.30.40",
			},
		},
	}

	kc := testutil.NewMockKubernetesClient(objs...)

	var validators []authenticate.Validator
	state := authenticate.NewInitializedAuthentication(&validators, &[]authenticate.Injector{})

	var namespaceUID string
	im := authenticate.NewKubernetesIntermediary(kc, net.ParseIP("10.20.30.40")).Chain(func(ctx context.Context, raw authenticate.Raw, md *authenticate.KubernetesIntermediaryMetadata) (authenticate.Intermediary, error) {
		namespaceUID = string(md.NamespaceUID)
		return authenticate.Raw(fmt.Sprintf("%s-processed", string(raw))), nil
	})
	raw, err := im.Next(ctx, state)
	require.NoError(t, err)
	require.Equal(t, authenticate.Raw("my-auth-token-processed"), raw)
	require.NotEmpty(t, namespaceUID)

	require.True(t, len(validators) > 0, "missing validators")

	claims := &authenticate.Claims{
		Claims: &jwt.Claims{
			Subject: "my-test-subject",
		},
		KubernetesNamespaceUID: namespaceUID,
	}

	for i, validator := range validators {
		outcome, err := validator.Validate(ctx, claims)
		require.True(t, outcome, "validator %d", i)
		require.NoError(t, err, "validator %d", i)
	}
}

func TestKubernetesIntermediaryChainToVault(t *testing.T) {
	ctx := context.Background()

	testutil.WithTestVaultClient(t, func(vc *vaultapi.Client) {
		// Vault configuration:
		require.NoError(t, vc.Sys().Mount("transit-test", &vaultapi.MountInput{
			Type: "transit",
		}))

		_, err := vc.Logical().Write("transit-test/keys/metadata-api", map[string]interface{}{
			"derived": true,
		})
		require.NoError(t, err)

		// Encrypt the token for the pod.
		secret, err := vc.Logical().Write("transit-test/encrypt/metadata-api", map[string]interface{}{
			"plaintext": base64.StdEncoding.EncodeToString([]byte("my-auth-token")),
			"context":   base64.StdEncoding.EncodeToString([]byte("hello")),
		})
		require.NoError(t, err)

		encryptedToken, ok := secret.Data["ciphertext"].(string)
		require.True(t, ok, "ciphertext is not a string")
		require.NotEmpty(t, encryptedToken)

		// Kubernetes configuration:
		objs := []runtime.Object{
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "my-test-namespace",
				},
			},
			&corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "my-test-namespace",
					Name:      "pod-a",
					Annotations: map[string]string{
						authenticate.KubernetesTokenAnnotation:   encryptedToken,
						authenticate.KubernetesSubjectAnnotation: "my-test-subject",
					},
				},
				Status: corev1.PodStatus{
					PodIP: "10.20.30.40",
				},
			},
		}

		kc := testutil.NewMockKubernetesClient(objs...)

		// Test:
		im := authenticate.NewChainIntermediary(
			authenticate.NewKubernetesIntermediary(kc, net.ParseIP("10.20.30.40")),
			authenticate.ChainVaultTransitIntermediary(vc, "transit-test", "metadata-api", authenticate.VaultTransitIntermediaryWithContext("hello")),
		)
		raw, err := im.Next(ctx, authenticate.NewAuthentication())
		require.NoError(t, err)
		require.Equal(t, authenticate.Raw("my-auth-token"), raw)
	})
}
