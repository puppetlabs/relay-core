package testutil

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/google/uuid"
	"github.com/puppetlabs/leg/k8sutil/pkg/manifest"
	"github.com/puppetlabs/leg/timeutil/pkg/retry"
	pvpoolv1alpha1 "github.com/puppetlabs/pvpool/pkg/apis/pvpool.puppet.com/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	tekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	tektonfake "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	cachingv1alpha1 "knative.dev/caching/pkg/apis/caching/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	TestScheme = runtime.NewScheme()
)

func init() {
	schemeBuilder := runtime.NewSchemeBuilder(
		dependency.AddToScheme,
		apiextensionsv1.AddToScheme,
		apiextensionsv1beta1.AddToScheme,
		cachingv1alpha1.AddToScheme,
		pvpoolv1alpha1.AddToScheme,
	)

	if err := schemeBuilder.AddToScheme(TestScheme); err != nil {
		panic(err)
	}
}

func NewMockKubernetesClient(initial ...runtime.Object) kubernetes.Interface {
	for _, obj := range initial {
		setObjectUIDOnObject(obj)
	}

	kc := fake.NewSimpleClientset(initial...)
	kc.PrependReactor("create", "*", setObjectUID)
	kc.PrependReactor("list", "pods", filterListPods(kc.Tracker()))
	return kc
}

func NewMockTektonKubernetesClient(initial ...runtime.Object) tekton.Interface {
	for _, obj := range initial {
		setObjectUIDOnObject(obj)
	}

	tkc := tektonfake.NewSimpleClientset(initial...)
	tkc.PrependReactor("create", "*", setObjectUID)
	return tkc
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
			if !la.GetListRestrictions().Fields.Matches(fields.Set{
				"status.podIP": pod.Status.PodIP,
				"status.phase": string(pod.Status.Phase),
			}) {
				continue
			}

			pods.Items[keep] = pod
			keep++
		}

		pods.Items = pods.Items[:keep]
		return true, pods, nil
	}
}

func ParseApplyKubernetesManifest(ctx context.Context, cl client.Client, r io.ReadCloser, patchers ...manifest.PatcherFunc) ([]manifest.Object, error) {
	objs, err := manifest.Parse(TestScheme, r, patchers...)
	if err != nil {
		return nil, err
	}

	for i, obj := range objs {
		name := obj.GetName()
		if obj.GetNamespace() != "" {
			name = obj.GetNamespace() + "/" + name
		}
		log.Printf("... applying %s %s", obj.GetObjectKind().GroupVersionKind(), name)

		if err := cl.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("relay-e2e")); err != nil {
			return nil, fmt.Errorf("could not apply object #%d %T: %+v", i, obj, err)
		}
	}

	return objs, nil
}

func WaitForServicesToBeReady(ctx context.Context, cl client.Client, namespace string) error {
	err := retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		eps := &corev1.EndpointsList{}
		if err := cl.List(ctx, eps, client.InNamespace(namespace)); err != nil {
			return true, err
		}

		if len(eps.Items) == 0 {
			return false, fmt.Errorf("waiting for endpoints")
		}

		for _, ep := range eps.Items {
			log.Println("checking service", ep.Name)
			if len(ep.Subsets) == 0 {
				return false, fmt.Errorf("waiting for subsets")
			}

			for _, subset := range ep.Subsets {
				if len(subset.Addresses) == 0 {
					return false, fmt.Errorf("waiting for pod assignment")
				}
			}
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func WaitForObjectDeletion(ctx context.Context, cl client.Client, obj client.Object) error {
	key := client.ObjectKeyFromObject(obj)

	return retry.Wait(ctx, func(ctx context.Context) (bool, error) {
		if err := cl.Get(ctx, key, obj); errors.IsNotFound(err) {
			return true, nil
		} else if err != nil {
			return true, err
		}

		return false, fmt.Errorf("waiting for deletion of %T %s", obj, key)
	})
}

func SetKubernetesEnvVar(target *[]corev1.EnvVar, name, value string) {
	for i, ev := range *target {
		if ev.Name == name {
			(*target)[i].Value = value
			return
		}
	}

	*target = append(*target, corev1.EnvVar{Name: name, Value: value})
}
