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
	WorkflowRunStateCancel = "cancel"
)

type WorkflowRun struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *relayv1beta1.Run
}

func makeWorkflowRun(key client.ObjectKey, obj *relayv1beta1.Run) *WorkflowRun {
	wr := &WorkflowRun{Key: key, Object: obj}
	wr.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&wr.Key, lifecycle.TypedObject{GVK: relayv1beta1.RunKind, Object: wr.Object})
	return wr
}

func (wr *WorkflowRun) Copy() *WorkflowRun {
	return makeWorkflowRun(wr.Key, wr.Object.DeepCopy())
}

func (wr *WorkflowRun) PersistStatus(ctx context.Context, cl client.Client) error {
	return cl.Status().Update(ctx, wr.Object)
}

func (wr *WorkflowRun) PodSelector() metav1.LabelSelector {
	return metav1.LabelSelector{
		MatchLabels: map[string]string{
			model.RelayControllerWorkflowRunIDLabel: wr.Key.Name,
		},
	}
}

func (wr *WorkflowRun) IsCancelled() bool {
	state, found := wr.Object.Spec.State.Workflow[WorkflowRunStateCancel]
	if !found {
		return false
	}

	return state.Value() == true
}

func NewWorkflowRun(key client.ObjectKey) *WorkflowRun {
	return makeWorkflowRun(key, &relayv1beta1.Run{})
}

func NewWorkflowRunFromObject(obj *relayv1beta1.Run) *WorkflowRun {
	return makeWorkflowRun(client.ObjectKeyFromObject(obj), obj)
}

func NewWorkflowRunPatcher(upd, orig *WorkflowRun) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
