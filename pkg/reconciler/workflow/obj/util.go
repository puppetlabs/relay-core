package obj

import (
	"context"
	"fmt"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ManagedByLabelValue = "relay.sh"
)

func GetIgnoreNotFound(ctx context.Context, cl client.Client, key client.ObjectKey, obj runtime.Object) (bool, error) {
	if err := cl.Get(ctx, key, obj); errors.IsNotFound(err) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return false, err
		}

		accessor.SetNamespace(key.Namespace)
		accessor.SetName(key.Name)

		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func CreateOrUpdate(ctx context.Context, cl client.Client, key client.ObjectKey, obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	accessor.SetNamespace(key.Namespace)
	accessor.SetName(key.Name)

	if len(accessor.GetUID()) == 0 {
		klog.Infof("creating %T %s", obj, key)
		return cl.Create(ctx, obj)
	}

	klog.Info("updating %T %s", obj, key)
	return cl.Update(ctx, obj)
}

func Annotate(target *metav1.ObjectMeta, name, value string) {
	if target.Annotations == nil {
		target.Annotations = make(map[string]string)
	}

	target.Annotations[name] = value
}

func Label(target *metav1.ObjectMeta, name, value string) {
	if target.Labels == nil {
		target.Labels = make(map[string]string)
	}

	target.Labels[name] = value
}

func CopyLabelsAndAnnotations(target *metav1.ObjectMeta, src metav1.ObjectMeta) {
	for name, value := range src.GetAnnotations() {
		Annotate(target, name, value)
	}

	for name, value := range src.GetLabels() {
		Label(target, name, value)
	}
}

func Own(target *metav1.ObjectMeta, ref *metav1.OwnerReference) {
	for _, c := range target.OwnerReferences {
		if equality.Semantic.DeepEqual(c, *ref) {
			return
		}
	}

	target.OwnerReferences = append(target.OwnerReferences, *ref)
	Label(target, "app.kubernetes.io/managed-by", ManagedByLabelValue)
}

func ModelStepFromName(wr *WorkflowRun, stepName string) *model.Step {
	return &model.Step{
		Run:  model.Run{ID: wr.Key.Name},
		Name: stepName,
	}
}

func ModelStep(wr *WorkflowRun, step *nebulav1.WorkflowStep) *model.Step {
	return ModelStepFromName(wr, step.Name)
}

func SuffixObjectKey(key client.ObjectKey, suffix string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      fmt.Sprintf("%s-%s", key.Name, suffix),
	}
}

func ModelStepObjectKey(key client.ObjectKey, ms *model.Step) client.ObjectKey {
	return SuffixObjectKey(key, ms.Hash().HexEncoding())
}
