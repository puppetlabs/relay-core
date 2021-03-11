package admission_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/operator/admission"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestPodEnforcementHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	testutil.WithEndToEndEnvironment(t, ctx, nil, func(e2e *testutil.EndToEndEnvironment) {
		mgr, err := manager.New(e2e.RESTConfig, manager.Options{
			Scheme:             testutil.TestScheme,
			MetricsBindAddress: "0",
		})
		require.NoError(t, err)

		testutil.WithPodEnforcementAdmissionRegistration(t, ctx, e2e, mgr, nil, nil, func() {
			e2e.WithTestNamespace(ctx, func(ns *corev1.Namespace) {
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
				require.NoError(t, e2e.ControllerClient.Create(ctx, pod))

				assert.Equal(t, admission.PodNodeSelector, pod.Spec.NodeSelector)
				assert.Equal(t, admission.PodTolerations, pod.Spec.Tolerations)
				assert.Equal(t, admission.PodDNSPolicy, pod.Spec.DNSPolicy)
				assert.Equal(t, admission.PodDNSConfig, pod.Spec.DNSConfig)
			})
		})
	})
}
