package task

import (
	"context"
	"fmt"

	"github.com/puppetlabs/nebula-tasks/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodMetadataManagerOptions struct {
	PodIP     string
	Namespace string
}

// PodMetadataManager provides metadata about a task by introspecting
// the Kubernetes pod it runs in using regular resource apis and kube
// clients.
type PodMetadataManager struct {
	kubeclient kubernetes.Interface
	options    PodMetadataManagerOptions
	pod        *corev1.Pod
}

func (mm *PodMetadataManager) Get(ctx context.Context) (*Metadata, errors.Error) {
	if mm.pod == nil {
		listOpts := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("status.podIP=%s", mm.options.PodIP),
		}

		pods, err := mm.kubeclient.CoreV1().Pods(mm.options.Namespace).List(listOpts)
		if err != nil {
			return nil, errors.NewKubernetesPodLookupError().WithCause(err)
		}

		if len(pods.Items) < 1 {
			return nil, errors.NewKubernetesPodNotFound(mm.options.PodIP)
		}

		// TODO fine tune this: this should theoretically never return more than 1 pod (if it does,
		// then out network fabric has some serious issues), but we should figure out how to handle
		// this scenario.
		pod := pods.Items[0]

		mm.pod = &pod
	}

	labels := mm.pod.GetLabels()

	return &Metadata{Name: labels["task-name"], ID: labels["task-ip"]}, nil
}

func NewPodMetadataManager(kubeclient kubernetes.Interface, options PodMetadataManagerOptions) *PodMetadataManager {
	return &PodMetadataManager{
		kubeclient: kubeclient,
		options:    options,
	}
}
