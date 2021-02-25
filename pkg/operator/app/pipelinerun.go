package app

import (
	"context"

	"github.com/puppetlabs/relay-core/pkg/model"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigurePipelineRun(ctx context.Context, pr *PipelineRun) error {
	if err := pr.Pipeline.Deps.WorkflowRun.Own(ctx, pr); err != nil {
		return err
	}

	pr.Label(ctx, model.RelayControllerWorkflowRunIDLabel, pr.Pipeline.Deps.WorkflowRun.Key.Name)

	sans := make([]tektonv1beta1.PipelineRunSpecServiceAccountName, len(pr.Pipeline.Object.Spec.Tasks))
	for i, pt := range pr.Pipeline.Object.Spec.Tasks {
		sans[i] = tektonv1beta1.PipelineRunSpecServiceAccountName{
			TaskName: pt.Name,
		}
	}

	pr.Object.Spec = tektonv1beta1.PipelineRunSpec{
		ServiceAccountNames: sans,
		PipelineRef: &tektonv1beta1.PipelineRef{
			Name: pr.Pipeline.Key.Name,
		},
	}

	if pr.Pipeline.Deps.WorkflowRun.IsCancelled() {
		pr.Object.Spec.Status = tektonv1beta1.PipelineRunSpecStatusCancelled
	}

	return nil
}

func ApplyPipelineRun(ctx context.Context, cl client.Client, p *Pipeline) (*PipelineRun, error) {
	pr := NewPipelineRun(p)

	if _, err := pr.Load(ctx, cl); err != nil {
		return nil, err
	}

	pr.LabelAnnotateFrom(ctx, p.Object.ObjectMeta)

	if err := ConfigurePipelineRun(ctx, pr); err != nil {
		return nil, err
	}

	if err := pr.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return pr, nil
}
