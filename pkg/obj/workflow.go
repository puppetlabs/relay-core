package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Workflow struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *relayv1beta1.Workflow
}

func makeWorkflow(key client.ObjectKey, obj *relayv1beta1.Workflow) *Workflow {
	w := &Workflow{Key: key, Object: obj}
	w.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&w.Key, lifecycle.TypedObject{GVK: relayv1beta1.WorkflowKind, Object: w.Object})
	return w
}

func (wr *Workflow) Copy() *Workflow {
	return makeWorkflow(wr.Key, wr.Object.DeepCopy())
}

func NewWorkflow(key client.ObjectKey) *Workflow {
	return makeWorkflow(key, &relayv1beta1.Workflow{})
}

func NewWorkflowFromObject(obj *relayv1beta1.Workflow) *Workflow {
	return makeWorkflow(client.ObjectKeyFromObject(obj), obj)
}
