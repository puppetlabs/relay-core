package obj

import (
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	PipelineKind = tektonv1beta1.SchemeGroupVersion.WithKind("Pipeline")
)

type Pipeline struct {
	*helper.NamespaceScopedAPIObject

	Key    client.ObjectKey
	Object *tektonv1beta1.Pipeline
}

func makePipeline(key client.ObjectKey, obj *tektonv1beta1.Pipeline) *Pipeline {
	p := &Pipeline{Key: key, Object: obj}
	p.NamespaceScopedAPIObject = helper.ForNamespaceScopedAPIObject(&p.Key, lifecycle.TypedObject{GVK: PipelineKind, Object: p.Object})
	return p
}

func (p *Pipeline) Copy() *Pipeline {
	return makePipeline(p.Key, p.Object.DeepCopy())
}

func (p *Pipeline) SetWorkspace(spec tektonv1beta1.PipelineWorkspaceDeclaration) {
	for i := range p.Object.Spec.Workspaces {
		ws := &p.Object.Spec.Workspaces[i]

		if ws.Name == spec.Name {
			*ws = spec
			return
		}
	}

	p.Object.Spec.Workspaces = append(p.Object.Spec.Workspaces, spec)
}

func NewPipeline(key client.ObjectKey) *Pipeline {
	return makePipeline(key, &tektonv1beta1.Pipeline{})
}

func NewPipelineFromObject(obj *tektonv1beta1.Pipeline) *Pipeline {
	return makePipeline(client.ObjectKeyFromObject(obj), obj)
}

func NewPipelinePatcher(upd, orig *Pipeline) lifecycle.Persister {
	return helper.NewPatcher(upd.Object, orig.Object, helper.WithObjectKey(upd.Key))
}
