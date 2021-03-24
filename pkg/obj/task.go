package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	TaskKind = tektonv1beta1.SchemeGroupVersion.WithKind("Task")
)

type Task struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *tektonv1beta1.Task
}

func makeTask(key client.ObjectKey, obj *tektonv1beta1.Task) *Task {
	t := &Task{Key: key, Object: obj}
	t.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&t.Key, lifecycle.TypedObject{GVK: TaskKind, Object: t.Object})
	return t
}

func (t *Task) Copy() *Task {
	return makeTask(t.Key, t.Object.DeepCopy())
}

func (t *Task) SetVolume(spec corev1.Volume) {
	for i := range t.Object.Spec.Volumes {
		v := &t.Object.Spec.Volumes[i]

		if v.Name == spec.Name {
			*v = spec
			return
		}
	}

	t.Object.Spec.Volumes = append(t.Object.Spec.Volumes, spec)
}

func (t *Task) SetWorkspace(spec tektonv1beta1.WorkspaceDeclaration) {
	for i := range t.Object.Spec.Workspaces {
		ws := &t.Object.Spec.Workspaces[i]

		if ws.Name == spec.Name {
			*ws = spec
			return
		}
	}

	t.Object.Spec.Workspaces = append(t.Object.Spec.Workspaces, spec)
}

func NewTask(key client.ObjectKey) *Task {
	return makeTask(key, &tektonv1beta1.Task{})
}

func NewTaskFromObject(obj *tektonv1beta1.Task) *Task {
	return makeTask(client.ObjectKeyFromObject(obj), obj)
}

func NewTaskPatcher(upd, orig *Task) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
