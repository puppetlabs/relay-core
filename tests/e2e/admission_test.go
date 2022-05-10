package e2e_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/operator/admission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodEnforcementHandler(t *testing.T) {
	t.Skip("to be replaced with policies enforced externally")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	WithNamespacedEnvironmentInTest(t, ctx, func(eit *EnvironmentInTest, ns *corev1.Namespace) {
		// Create a pod in the test namespace and check that the admission
		// controller applied our desired updates.
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.GetName(),
				Name:      "sneaky-pod",
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    "sneak",
						Image:   "alpine:latest",
						Command: []string{"sh", "-c", "trap : TERM INT; sleep 600 & wait"},
					},
				},
			},
		}
		require.NoError(t, eit.ControllerClient.Create(ctx, pod))

		assert.Equal(t, admission.PodNodeSelector, pod.Spec.NodeSelector)
		assert.Equal(t, admission.PodTolerations, pod.Spec.Tolerations)
		assert.Equal(t, admission.PodDNSPolicy, pod.Spec.DNSPolicy)
		assert.Equal(t, admission.PodDNSConfig, pod.Spec.DNSConfig)
	})
}
