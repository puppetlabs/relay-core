package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigurePipelineRun(ctx context.Context, pr *obj.PipelineRun, pp *PipelineParts) error {
	lifecycle.Label(ctx, pr, model.RelayControllerWorkflowRunIDLabel, pp.Deps.Run.Key.Name)
	pr.LabelAnnotateFrom(ctx, pp.Deps.Run.Object)

	if err := pp.Deps.OwnerConfigMap.Own(ctx, pr); err != nil {
		return err
	}

	if err := DependencyManager.SetDependencyOf(
		&pr.Object.ObjectMeta,
		lifecycle.TypedObject{
			Object: pp.Deps.Run.Object,
			GVK:    relayv1beta1.RunKind,
		}); err != nil {
		return err
	}

	sans := make([]tektonv1beta1.PipelineRunSpecServiceAccountName, len(pp.Pipeline.Object.Spec.Tasks))
	for i, pt := range pp.Pipeline.Object.Spec.Tasks {
		sans[i] = tektonv1beta1.PipelineRunSpecServiceAccountName{
			TaskName: pt.Name,
		}
	}

	pr.Object.Spec = tektonv1beta1.PipelineRunSpec{
		ServiceAccountNames: sans,
		PipelineRef: &tektonv1beta1.PipelineRef{
			Name: pp.Pipeline.Key.Name,
		},
		PodTemplate: &tektonv1beta1.PodTemplate{
			EnableServiceLinks: pointer.BoolPtr(false),
		},
	}

	pr.Object.Spec.Workspaces = []tektonv1beta1.WorkspaceBinding{
		{
			Name:     ToolsWorkspaceName,
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	if pp.Deps.Run.IsCancelled() {
		pr.Object.Spec.Status = tektonv1beta1.PipelineRunSpecStatusCancelled
	}

	return nil
}

func ApplyPipelineRun(ctx context.Context, cl client.Client, pp *PipelineParts) (*obj.PipelineRun, error) {
	pr := obj.NewPipelineRun(pp.Pipeline.Key)

	if _, err := pr.Load(ctx, cl); err != nil {
		return nil, err
	}

	pr.LabelAnnotateFrom(ctx, pp.Pipeline.Object)

	if err := ConfigurePipelineRun(ctx, pr, pp); err != nil {
		return nil, err
	}

	if err := pr.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return pr, nil
}
