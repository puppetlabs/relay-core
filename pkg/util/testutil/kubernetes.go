package testutil

import (
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
)

func NewMockKubernetesClient(initial ...runtime.Object) kubernetes.Interface {
	for _, obj := range initial {
		setObjectUIDOnObject(obj)
	}

	kc := fake.NewSimpleClientset(initial...)
	kc.PrependReactor("create", "*", setObjectUID)
	kc.PrependReactor("list", "pods", filterListPods(kc.Tracker()))
	return kc
}

func setObjectUID(action testing.Action) (bool, runtime.Object, error) {
	switch action := action.(type) {
	case testing.CreateActionImpl:
		obj := action.GetObject()
		setObjectUIDOnObject(obj)
		return false, obj, nil
	default:
		return false, nil, nil
	}
}

func setObjectUIDOnObject(obj runtime.Object) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return
	}

	accessor.SetUID(types.UID(uuid.New().String()))
}

func filterListPods(tracker testing.ObjectTracker) testing.ReactionFunc {
	delegate := testing.ObjectReaction(tracker)

	return func(action testing.Action) (bool, runtime.Object, error) {
		la := action.(testing.ListAction)

		found, obj, err := delegate(action)
		if err != nil || !found {
			return found, obj, err
		}

		pods := obj.(*corev1.PodList)

		keep := 0
		for _, pod := range pods.Items {
			if !la.GetListRestrictions().Fields.Matches(fields.Set{"status.podIP": pod.Status.PodIP}) {
				continue
			}

			pods.Items[keep] = pod
			keep++
		}

		pods.Items = pods.Items[:keep]
		return true, pods, nil
	}
}
