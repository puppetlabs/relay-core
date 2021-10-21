package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ToolsWorkspaceName = "tools"

type PipelineParts struct {
	Deps *RunDeps

	Tasks      *TaskSet
	Conditions *ConditionSet
	Pipeline   *obj.Pipeline
}

var _ lifecycle.LabelAnnotatableFrom = &PipelineParts{}
var _ lifecycle.Loader = &PipelineParts{}
var _ lifecycle.Ownable = &PipelineParts{}
var _ lifecycle.Persister = &PipelineParts{}

func (pp *PipelineParts) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	lafs := []lifecycle.LabelAnnotatableFrom{
		pp.Tasks,
		pp.Conditions,
		pp.Pipeline,
	}
	for _, laf := range lafs {
		laf.LabelAnnotateFrom(ctx, from)
	}
}

func (pp *PipelineParts) Load(ctx context.Context, cl client.Client) (bool, error) {
	return lifecycle.Loaders{
		pp.Tasks,
		pp.Conditions,
		pp.Pipeline,
	}.Load(ctx, cl)
}

func (pp *PipelineParts) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return lifecycle.OwnablePersisters{
		pp.Tasks,
		pp.Conditions,
		pp.Pipeline,
	}.Owned(ctx, owner)
}

func (pp *PipelineParts) Persist(ctx context.Context, cl client.Client) error {
	return lifecycle.OwnablePersisters{
		pp.Tasks,
		pp.Conditions,
		pp.Pipeline,
	}.Persist(ctx, cl)
}

func NewPipelineParts(deps *RunDeps) *PipelineParts {
	return &PipelineParts{
		Deps: deps,

		Tasks:      NewTaskSet(deps),
		Conditions: NewConditionSet(deps),
		Pipeline: obj.NewPipeline(
			client.ObjectKey{
				Namespace: deps.WorkflowDeps.TenantDeps.Namespace.Name,
				Name:      deps.WorkflowRun.Key.Name,
			},
		),
	}
}

func ConfigurePipelineParts(ctx context.Context, p *PipelineParts) error {
	p.Pipeline.LabelAnnotateFrom(ctx, p.Deps.WorkflowRun.Object)

	if err := p.Deps.OwnerConfigMap.Own(ctx, p); err != nil {
		return err
	}

	if err := DependencyManager.SetDependencyOf(
		&p.Pipeline.Object.ObjectMeta,
		lifecycle.TypedObject{
			Object: p.Deps.WorkflowRun.Object,
			GVK:    relayv1beta1.RunKind,
		}); err != nil {
		return err
	}

	if err := ConfigureConditionSet(ctx, p.Conditions); err != nil {
		return err
	}

	if err := ConfigureTaskSet(ctx, p.Tasks); err != nil {
		return err
	}

	if p.Deps.ToolInjectionCheckout.Satisfied() {
		p.Pipeline.SetWorkspace(tektonv1beta1.PipelineWorkspaceDeclaration{
			Name: ToolsWorkspaceName,
		})
	}

	p.Pipeline.Object.Spec.Tasks = make([]tektonv1beta1.PipelineTask, 0, len(p.Tasks.List))

	for i, t := range p.Tasks.List {
		ws := p.Deps.Workflow.Object.Spec.Steps[i]
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

		if p.Deps.ToolInjectionCheckout.Satisfied() {
			pt.Workspaces = []tektonv1beta1.WorkspacePipelineTaskBinding{
				{
					Name:      ToolsWorkspaceName,
					Workspace: ToolsWorkspaceName,
				},
			}
		}

		p.Pipeline.Object.Spec.Tasks = append(p.Pipeline.Object.Spec.Tasks, pt)
	}

	return nil
}

func ApplyPipelineParts(ctx context.Context, cl client.Client, deps *RunDeps) (*PipelineParts, error) {
	p := NewPipelineParts(deps)

	if _, err := p.Load(ctx, cl); err != nil {
		return nil, err
	}

	p.LabelAnnotateFrom(ctx, deps.WorkflowRun.Object)

	if err := ConfigurePipelineParts(ctx, p); err != nil {
		return nil, err
	}

	if err := p.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return p, nil
}
