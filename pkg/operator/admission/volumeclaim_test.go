package admission_test

import (
	"context"
	"testing"
	"time"

	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/util/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestVolumeClaimHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	testutil.WithEndToEndEnvironment(t, ctx, nil, func(e2e *testutil.EndToEndEnvironment) {
		mgr, err := manager.New(e2e.RESTConfig, manager.Options{
			Scheme:             testutil.TestScheme,
			MetricsBindAddress: "0",
		})
		require.NoError(t, err)

		testutil.WithVolumeClaimAdmissionRegistration(t, ctx, e2e, mgr, nil, nil, func() {
			e2e.WithTestNamespace(ctx, func(ns *corev1.Namespace) {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ns.GetName(),
						Name:      "sneaky-pod",
						Annotations: map[string]string{
							model.RelayControllerToolsVolumeClaimAnnotation: "tools-volume-claim",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:    "hide",
								Image:   "alpine:latest",
								Command: []string{"sh", "-c", "trap : TERM INT; sleep 600 & wait"},
							},
							{
								Name:    "sneak",
								Image:   "alpine:latest",
								Command: []string{"sh", "-c", "trap : TERM INT; sleep 600 & wait"},
							},
						},
					},
				}
				require.NoError(t, e2e.ControllerClient.Create(ctx, pod))

				volume := false
				for _, v := range pod.Spec.Volumes {
					if v.Name == model.ToolInjectionMountName &&
						v.VolumeSource.PersistentVolumeClaim.ClaimName == "tools-volume-claim" {
						volume = true
					}
				}

				assert.True(t, volume)

				volumeMount := false
				for _, c := range pod.Spec.Containers {
					for _, vm := range c.VolumeMounts {
						if vm.Name == model.ToolInjectionMountName {
							volumeMount = true
						}
					}

					assert.True(t, volumeMount)
				}
			})
		})
	})
}
