package obj

import (
	"context"

	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pipeline struct {
	Deps *PipelineDeps

	Key    client.ObjectKey
	Object *tektonv1beta1.Pipeline

	Tasks      *Tasks
	Conditions *Conditions
}

var _ Persister = &Pipeline{}
var _ Loader = &Pipeline{}
var _ Ownable = &Pipeline{}
var _ LabelAnnotatableFrom = &Pipeline{}

func (p *Pipeline) Persist(ctx context.Context, cl client.Client) error {
	if err := p.Tasks.Persist(ctx, cl); err != nil {
		return err
	}

	if err := p.Conditions.Persist(ctx, cl); err != nil {
		return err
	}

	return CreateOrUpdate(ctx, cl, p.Key, p.Object)
}

func (p *Pipeline) Load(ctx context.Context, cl client.Client) (bool, error) {
	all := true

	if ok, err := GetIgnoreNotFound(ctx, cl, p.Key, p.Object); err != nil {
		return false, err
	} else if !ok {
		all = false
	}

	if ok, err := (Loaders{p.Tasks, p.Conditions}).Load(ctx, cl); err != nil {
		return false, err
	} else if !ok {
		all = false
	}

	return all, nil
}

func (p *Pipeline) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&p.Object.ObjectMeta, ref)
}

func (p *Pipeline) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&p.Object.ObjectMeta, from)
}

func NewPipeline(pd *PipelineDeps) *Pipeline {
	return &Pipeline{
		Deps:   pd,
		Key:    pd.WorkflowRun.Key,
		Object: &tektonv1beta1.Pipeline{},

		Tasks:      NewTasks(pd),
		Conditions: NewConditions(pd),
	}
}

func ConfigurePipeline(ctx context.Context, p *Pipeline) error {
	if err := ConfigureConditions(ctx, p.Conditions); err != nil {
		return err
	}

	if err := ConfigureTasks(ctx, p.Tasks); err != nil {
		return err
	}

	p.Object.Spec.Tasks = make([]tektonv1beta1.PipelineTask, 0, len(p.Tasks.List))

	for i, t := range p.Tasks.List {
		ws := p.Deps.WorkflowRun.Object.Spec.Workflow.Steps[i]
		ms := ModelStep(p.Deps.WorkflowRun, ws)

		pt := tektonv1beta1.PipelineTask{
			Name: ms.Hash().HexEncoding(),
			TaskRef: &tektonv1beta1.TaskRef{
				Name: t.Key.Name,
			},
			RunAfter: make([]string, len(ws.DependsOn)),
		}

		for i, dep := range ws.DependsOn {
			pt.RunAfter[i] = ModelStepFromName(p.Deps.WorkflowRun, dep).Hash().HexEncoding()
		}

		if cond, ok := p.Conditions.GetByStepName(ws.Name); ok {
			pt.Conditions = []tektonv1beta1.PipelineTaskCondition{
				{ConditionRef: cond.Key.Name},
			}
		}

		p.Object.Spec.Tasks = append(p.Object.Spec.Tasks, pt)
	}

	return nil
}

func ApplyPipeline(ctx context.Context, cl client.Client, deps *PipelineDeps) (*Pipeline, error) {
	p := NewPipeline(deps)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	p.LabelAnnotateFrom(ctx, deps.WorkflowRun.Object.ObjectMeta)

	if err := ConfigurePipeline(ctx, p); err != nil {
		return nil, err
	}

	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}
