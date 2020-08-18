package obj

import (
	"context"
	"encoding/json"
	"fmt"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ManagedByLabelValue = "relay.sh"
)

type OwnerInOtherNamespaceError struct {
	Owner     Owner
	OwnerKey  client.ObjectKey
	Target    runtime.Object
	TargetKey client.ObjectKey
}

func (e *OwnerInOtherNamespaceError) Error() string {
	return fmt.Sprintf("obj: owner %T %s is in a different namespace than %T %s", e.Owner.Object, e.OwnerKey, e.Target, e.TargetKey)
}

func Create(ctx context.Context, cl client.Client, key client.ObjectKey, obj runtime.Object) error {
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

	return nil
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

	klog.Infof("updating %T %s", obj, key)
	return cl.Update(ctx, obj)
}

func GetIgnoreNotFound(ctx context.Context, cl client.Client, key client.ObjectKey, obj runtime.Object) (bool, error) {
	if err := cl.Get(ctx, key, obj); k8serrors.IsNotFound(err) {
		klog.V(2).Infof("object %T %s not found", obj, key)

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

func DeleteIgnoreNotFound(ctx context.Context, cl client.Client, obj runtime.Object) (bool, error) {
	if err := cl.Delete(ctx, obj); k8serrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func AddFinalizer(target *metav1.ObjectMeta, name string) bool {
	for _, f := range target.Finalizers {
		if f == name {
			return false
		}
	}

	target.Finalizers = append(target.Finalizers, name)
	return true
}

func RemoveFinalizer(target *metav1.ObjectMeta, name string) bool {
	cut := -1
	for i, f := range target.Finalizers {
		if f == name {
			cut = i
			break
		}
	}

	if cut < 0 {
		return false
	}

	target.Finalizers[cut] = target.Finalizers[len(target.Finalizers)-1]
	target.Finalizers = target.Finalizers[:len(target.Finalizers)-1]
	return true
}

type DependencyOf struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Namespace  string    `json:"namespace,omitempty"`
	Name       string    `json:"name"`
	UID        types.UID `json:"uid"`
}

func SetDependencyOf(target *metav1.ObjectMeta, owner Owner) error {
	accessor, err := meta.Accessor(owner.Object)
	if err != nil {
		return err
	}

	annotation, err := json.Marshal(DependencyOf{
		APIVersion: owner.GVK.GroupVersion().Identifier(),
		Kind:       owner.GVK.Kind,
		Namespace:  accessor.GetNamespace(),
		Name:       accessor.GetName(),
		UID:        accessor.GetUID(),
	})
	if err != nil {
		return err
	}

	Annotate(target, model.RelayControllerDependencyOfAnnotation, string(annotation))
	return nil
}

func IsDependencyOf(target metav1.ObjectMeta, owner Owner) (bool, error) {
	annotation := target.GetAnnotations()[model.RelayControllerDependencyOfAnnotation]
	if annotation == "" {
		return false, nil
	}

	var dep DependencyOf
	if err := json.Unmarshal([]byte(annotation), &dep); err != nil {
		return false, err
	}

	depGroupVersion, _ := schema.ParseGroupVersion(dep.APIVersion)

	if owner.GVK.Kind != dep.Kind || owner.GVK.GroupVersion() != depGroupVersion {
		return false, nil
	}

	accessor, err := meta.Accessor(owner.Object)
	if err != nil {
		return false, err
	}

	return accessor.GetUID() == dep.UID && accessor.GetNamespace() == dep.Namespace && accessor.GetName() == dep.Name, nil
}

func Annotate(target *metav1.ObjectMeta, name, value string) bool {
	if target.Annotations == nil {
		target.Annotations = make(map[string]string)
	} else if candidate, ok := target.Annotations[name]; ok && candidate == value {
		return false
	}

	target.Annotations[name] = value
	return true
}

func Label(target *metav1.ObjectMeta, name, value string) bool {
	if target.Labels == nil {
		target.Labels = make(map[string]string)
	} else if candidate, ok := target.Labels[name]; ok && candidate == value {
		return false
	}

	target.Labels[name] = value
	return true
}

func CopyLabelsAndAnnotations(target *metav1.ObjectMeta, src metav1.ObjectMeta) {
	for name, value := range src.GetAnnotations() {
		Annotate(target, name, value)
	}

	for name, value := range src.GetLabels() {
		Label(target, name, value)
	}
}

func Own(target runtime.Object, owner Owner) error {
	targetAccessor, err := meta.Accessor(target)
	if err != nil {
		return err
	}

	ownerAccessor, err := meta.Accessor(owner.Object)
	if err != nil {
		return err
	}

	if targetAccessor.GetNamespace() != ownerAccessor.GetNamespace() {
		return &OwnerInOtherNamespaceError{
			Owner:     owner,
			OwnerKey:  client.ObjectKey{Namespace: ownerAccessor.GetNamespace(), Name: ownerAccessor.GetName()},
			Target:    target,
			TargetKey: client.ObjectKey{Namespace: targetAccessor.GetNamespace(), Name: targetAccessor.GetName()},
		}
	}

	targetLabels := targetAccessor.GetLabels()
	if targetLabels == nil {
		targetLabels = make(map[string]string)
	}
	targetLabels["app.kubernetes.io/managed-by"] = ManagedByLabelValue
	targetAccessor.SetLabels(targetLabels)

	ref := metav1.NewControllerRef(ownerAccessor, owner.GVK)

	targetOwners := targetAccessor.GetOwnerReferences()
	for i, c := range targetOwners {
		if equality.Semantic.DeepEqual(c, *ref) {
			return nil
		} else if c.Controller != nil && *c.Controller == true {
			c.Controller = func(b bool) *bool { return &b }(false)
			klog.Warningf(
				"%T %s/%s is stealing controller for %T %s/%s from %s %s/%s",
				owner.Object, ownerAccessor.GetNamespace(), ownerAccessor.GetName(),
				target, targetAccessor.GetNamespace(), targetAccessor.GetName(),
				c.Kind, targetAccessor.GetNamespace(), c.Name,
			)

			targetOwners[i] = c
		}
	}

	targetOwners = append(targetOwners, *ref)
	targetAccessor.SetOwnerReferences(targetOwners)

	return nil
}

func ModelStepFromName(wr *WorkflowRun, stepName string) *model.Step {
	return &model.Step{
		Run:  model.Run{ID: wr.Object.Spec.Name},
		Name: stepName,
	}
}

func ModelStep(wr *WorkflowRun, step *nebulav1.WorkflowStep) *model.Step {
	return ModelStepFromName(wr, step.Name)
}

func ModelWebhookTrigger(wt *WebhookTrigger) *model.Trigger {
	name := wt.Object.Spec.Name
	if name == "" {
		name = wt.Key.Name
	}

	return &model.Trigger{
		Name: name,
	}
}

func SuffixObjectKey(key client.ObjectKey, suffix string) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      fmt.Sprintf("%s-%s", key.Name, suffix),
	}
}

func ModelStepObjectKey(key client.ObjectKey, ms *model.Step) client.ObjectKey {
	return client.ObjectKey{
		Namespace: key.Namespace,
		Name:      ms.Hash().HexEncoding(),
	}
}
