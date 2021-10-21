package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RunStateCancel = "cancel"
)

type Run struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *relayv1beta1.Run
}

func makeRun(key client.ObjectKey, obj *relayv1beta1.Run) *Run {
	r := &Run{Key: key, Object: obj}
	r.NamespaceScopedAPIObject =
		helper.ForNamespaceScopedAPIObject(
			&r.Key,
			lifecycle.TypedObject{
				GVK:    relayv1beta1.RunKind,
				Object: r.Object,
			},
		)
	return r
}

func (r *Run) Copy() *Run {
	return makeRun(r.Key, r.Object.DeepCopy())
}

func (r *Run) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, r.Object)
}

func (r *Run) PodSelector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			model.RelayControllerWorkflowRunIDLabel: r.Key.Name,
		},
	}
}

func (r *Run) IsCancelled() bool {
	state, found := r.Object.Spec.State.Workflow[RunStateCancel]
	if !found {
		return false
	}

	return state.Value() == true
}

func NewRun(key client.ObjectKey) *Run {
	return makeRun(key, &relayv1beta1.Run{})
}

func NewRunFromObject(obj *relayv1beta1.Run) *Run {
	return makeRun(client.ObjectKeyFromObject(obj), obj)
}

func NewRunPatcher(upd, orig *Run) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
