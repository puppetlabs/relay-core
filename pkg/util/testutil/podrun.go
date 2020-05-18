package testutil

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/puppetlabs/nebula-tasks/pkg/util/retry"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/exec"
)

func RunScriptInAlpine(t *testing.T, ctx context.Context, cfg *rest.Config, ifc kubernetes.Interface, meta metav1.ObjectMeta, script string) (int, string, string) {
	if meta.GetName() == "" && meta.GetGenerateName() == "" {
		meta.SetGenerateName("script-")
	}

	pod := &corev1.Pod{
		ObjectMeta: meta,
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "script",
					Image: "alpine:latest",
					Args: []string{
						"/bin/sh", "-c", "trap : TERM INT; sleep 86400 & wait",
					},
				},
			},
		},
	}

	pc := ifc.CoreV1().Pods(pod.GetNamespace())

	pod, err := pc.Create(pod)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, pc.Delete(pod.GetName(), &metav1.DeleteOptions{}))
	}()

	// Wait for pod ready.
	require.NoError(t, retry.Retry(ctx, 500*time.Millisecond, func() *retry.RetryError {
		pod, err = pc.Get(pod.GetName(), metav1.GetOptions{})
		require.NoError(t, err)

		switch pod.Status.Phase {
		case corev1.PodFailed, corev1.PodSucceeded:
			return retry.RetryPermanent(fmt.Errorf("pod terminated unexpectedly"))
		case corev1.PodRunning:
			return retry.RetryPermanent(nil)
		}

		return retry.RetryTransient(fmt.Errorf("waiting for pod to start"))
	}))

	// Exec into pod.
	req := ifc.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(pod.GetNamespace()).
		Name(pod.GetName()).
		SubResource("exec").
		Param("container", "script")
	req.VersionedParams(&v1.PodExecOptions{
		Container: "script",
		Command:   []string{"/bin/sh", "-c", script},
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	execer, err := remotecommand.NewSPDYExecutor(cfg, http.MethodPost, req.URL())
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	err = execer.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	var code int
	if cerr, ok := err.(exec.CodeExitError); ok {
		code = cerr.ExitStatus()
	} else {
		require.NoError(t, err)
	}

	return code, stdout.String(), stderr.String()
}
