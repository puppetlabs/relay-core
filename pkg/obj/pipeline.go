package obj

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pipeline struct {
	Key    client.ObjectKey
	Object *tektonv1beta1.Pipeline
}

var _ lifecycle.LabelAnnotatableFrom = &Pipeline{}
var _ lifecycle.Loader = &Pipeline{}
var _ lifecycle.Ownable = &Pipeline{}
var _ lifecycle.Persister = &Pipeline{}

func (p *Pipeline) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	helper.CopyLabelsAndAnnotations(&p.Object.ObjectMeta, from)
}

func (p *Pipeline) Load(ctx context.Context, cl client.Client) (bool, error) {
	return helper.GetIgnoreNotFound(ctx, cl, p.Key, p.Object)
}

func (p *Pipeline) Persist(ctx context.Context, cl client.Client) error {
	return helper.CreateOrUpdate(ctx, cl, p.Object, helper.WithObjectKey(p.Key))
}

func (p *Pipeline) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(p.Object, owner)
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
	return &Pipeline{
		Key:    key,
		Object: &tektonv1beta1.Pipeline{},
	}
}
