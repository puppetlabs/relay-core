package testutil

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/puppetlabs/relay-core/pkg/util/retry"
	"github.com/rancher/remotedialer"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	BindPodTunnelImage = "inlets/inlets:2.7.3"
	BindPodProxyImage  = "squareup/ghostunnel:v1.5.2"
)

func WithServiceBoundToHostHTTP(t *testing.T, ctx context.Context, cfg *rest.Config, ifc kubernetes.Interface, targetURL string, meta metav1.ObjectMeta, fn func(caPEM []byte, svc *corev1.Service)) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Phase 1: Create a server that can forward connections for us.
	if meta.GetName() == "" && meta.GetGenerateName() == "" {
		meta.SetGenerateName(fmt.Sprintf("bind-%s-", strings.Map(func(r rune) rune {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}

			return '-'
		}, targetURL)))
	}

	if meta.Labels == nil {
		meta.Labels = make(map[string]string)
	}

	id := uuid.New().String()
	meta.Labels["testing.relay.sh/bind-selector"] = id

	// Create secret with TLS information.
	tls := GenerateCertificateBundle(t, fmt.Sprintf("bind-%s.%s.svc", id, meta.GetNamespace()))

	secret := &corev1.Secret{
		ObjectMeta: meta,
		Type:       corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.key": tls.ServerKeyPEM,
			"tls.crt": tls.BundlePEM,
		},
	}
	secret, err := ifc.CoreV1().Secrets(secret.GetNamespace()).Create(secret)
	require.NoError(t, err)

	// Create pod to provide tunneling.
	pod := &corev1.Pod{
		ObjectMeta: meta,
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "tunnel",
					Image: BindPodTunnelImage,
					Args: []string{
						"server",
						"--port", "8000",
						"--control-port", "8080",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "proxy-http",
							ContainerPort: 8000,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "tunnel",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
					},
				},
				{
					Name:  "proxy",
					Image: BindPodProxyImage,
					Args: []string{
						"server",
						"--listen", ":9000",
						"--target", "localhost:8000",
						"--disable-authentication",
						"--key", "/var/run/secrets/testing.relay.sh/tls/tls.key",
						"--cert", "/var/run/secrets/testing.relay.sh/tls/tls.crt",
						"--status", ":9001",
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "tls",
							MountPath: "/var/run/secrets/testing.relay.sh/tls",
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "proxy-https",
							ContainerPort: 9000,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "proxy-status",
							ContainerPort: 9001,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/_status",
								Port:   intstr.FromString("proxy-status"),
								Scheme: corev1.URISchemeHTTPS,
							},
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "tls",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secret.GetName(),
						},
					},
				},
			},
		},
	}
	pod, err = ifc.CoreV1().Pods(pod.GetNamespace()).Create(pod)
	require.NoError(t, err)

	// Forward pod to establish local side.
	req := ifc.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.GetNamespace()).
		Name(pod.GetName()).
		SubResource("portforward").
		Param("ports", "8080")

	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	require.NoError(t, err)

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	readyCh := make(chan struct{})

	pf, err := portforward.New(dialer, []string{":8080"}, ctx.Done(), readyCh, os.Stdout, os.Stderr)
	require.NoError(t, err)
	defer pf.Close()

	go func() {
		retry.Retry(ctx, 1*time.Second, func() *retry.RetryError {
			if err := pf.ForwardPorts(); err != nil {
				return retry.RetryTransient(err)
			}

			return retry.RetryPermanent(nil)
		})
	}()

	select {
	case <-readyCh:
	case <-ctx.Done():
		require.FailNow(t, "timed out waiting for port forwarder")
	}

	// Connect local side to serving address.
	ports, err := pf.GetPorts()
	require.NoError(t, err)
	require.Len(t, ports, 1)

	go func() {
		headers := make(http.Header)
		headers.Set("x-inlets-id", id)
		headers.Set("x-inlets-upstream", "="+targetURL)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			remotedialer.ClientConnect(
				ctx,
				fmt.Sprintf("ws://localhost:%d/tunnel", ports[0].Local),
				headers,
				nil,
				func(proto, address string) bool { return true },
				nil,
			)
		}
	}()

	// Create and wait for service.
	svc := &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name:       "proxy-http",
					TargetPort: intstr.FromString("proxy-http"),
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
				},
				{
					Name:       "proxy-https",
					TargetPort: intstr.FromString("proxy-https"),
					Protocol:   corev1.ProtocolTCP,
					Port:       443,
				},
			},
			Selector: map[string]string{
				"testing.relay.sh/bind-selector": id,
			},
		},
	}
	svc.SetName(fmt.Sprintf("bind-%s", id))
	svc, err = ifc.CoreV1().Services(svc.GetNamespace()).Create(svc)
	require.NoError(t, err)

	require.NoError(t, retry.Retry(ctx, 2*time.Second, func() *retry.RetryError {
		ep, err := ifc.CoreV1().Endpoints(svc.GetNamespace()).Get(svc.GetName(), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return retry.RetryTransient(fmt.Errorf("waiting for endpoints"))
		} else if err != nil {
			return retry.RetryPermanent(err)
		}

		if len(ep.Subsets) != 1 {
			return retry.RetryTransient(fmt.Errorf("waiting for subsets"))
		}

		if len(ep.Subsets[0].Addresses) == 0 {
			return retry.RetryTransient(fmt.Errorf("waiting for pod assignment"))
		}

		return retry.RetryPermanent(nil)
	}))

	fn(tls.AuthorityPEM, svc)
}
