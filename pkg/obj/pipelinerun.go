package obj

import (
	"context"

	"github.com/puppetlabs/nebula-tasks/pkg/model"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	PipelineRunPodNodeSelector = map[string]string{
		"nebula.puppet.com/scheduling.customer-ready": "true",
	}
	PipelineRunPodTolerations = []corev1.Toleration{
		{
			Key:    "nebula.puppet.com/scheduling.customer-workload",
			Value:  "true",
			Effect: corev1.TaintEffectNoSchedule,
		},
	}
	PipelineRunPodDNSPolicy = func(p corev1.DNSPolicy) *corev1.DNSPolicy { return &p }(corev1.DNSNone)
	PipelineRunPodDNSConfig = &corev1.PodDNSConfig{
		Nameservers: []string{
			"1.1.1.1",
			"1.0.0.1",
			"8.8.8.8",
		},
	}
)

type PipelineRun struct {
	Pipeline *Pipeline

	Key    client.ObjectKey
	Object *tektonv1beta1.PipelineRun
}

var _ Persister = &PipelineRun{}
var _ Loader = &PipelineRun{}
var _ Ownable = &PipelineRun{}
var _ LabelAnnotatableFrom = &PipelineRun{}

func (pr *PipelineRun) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, pr.Key, pr.Object)
}

func (pr *PipelineRun) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, pr.Key, pr.Object)
}

func (pr *PipelineRun) Owned(ctx context.Context, ref *metav1.OwnerReference) {
	Own(&pr.Object.ObjectMeta, ref)
}

func (pr *PipelineRun) Label(ctx context.Context, name, value string) {
	Label(&pr.Object.ObjectMeta, name, value)
}

func (pr *PipelineRun) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&pr.Object.ObjectMeta, from)
}

func (pr *PipelineRun) IsComplete() bool {
	if !pr.Object.IsDone() && !pr.Object.IsCancelled() && !pr.Object.IsTimedOut() {
		return false
	}

	for _, tr := range pr.Object.Status.TaskRuns {
		if tr.Status == nil {
			continue
		}

		if workflowRunStatus(tr.Status.Status) == WorkflowRunStatusInProgress {
			return false
		}
	}

	return true
}

func NewPipelineRun(p *Pipeline) *PipelineRun {
	return &PipelineRun{
		Pipeline: p,

		Key:    p.Key,
		Object: &tektonv1beta1.PipelineRun{},
	}
}

func ConfigurePipelineRun(ctx context.Context, pr *PipelineRun) {
	pr.Pipeline.Deps.WorkflowRun.Own(ctx, pr)
	pr.Label(ctx, model.RelayControllerWorkflowRunIDLabel, pr.Pipeline.Deps.WorkflowRun.Key.Name)

	sans := make([]tektonv1beta1.PipelineRunSpecServiceAccountName, len(pr.Pipeline.Object.Spec.Tasks))
	for i, pt := range pr.Pipeline.Object.Spec.Tasks {
		sans[i] = tektonv1beta1.PipelineRunSpecServiceAccountName{
			TaskName:           pt.Name,
			ServiceAccountName: pr.Pipeline.Deps.PipelineServiceAccount.Key.Name,
		}
	}

	pr.Object.Spec = tektonv1beta1.PipelineRunSpec{
		ServiceAccountName:  pr.Pipeline.Deps.SystemServiceAccount.Key.Name,
		ServiceAccountNames: sans,
		PipelineRef: &tektonv1beta1.PipelineRef{
			Name: pr.Pipeline.Key.Name,
		},
		PodTemplate: &tektonv1beta1.PodTemplate{
			NodeSelector: PipelineRunPodNodeSelector,
			Tolerations:  PipelineRunPodTolerations,
			DNSPolicy:    PipelineRunPodDNSPolicy,
			DNSConfig:    PipelineRunPodDNSConfig,
		},
	}

	if pr.Pipeline.Deps.WorkflowRun.IsCancelled() {
		pr.Object.Spec.Status = tektonv1beta1.PipelineRunSpecStatusCancelled
	}
}

func ApplyPipelineRun(ctx context.Context, cl client.Client, p *Pipeline) (*PipelineRun, error) {
	pr := NewPipelineRun(p)

	if _, err := pr.Load(ctx, cl); err != nil {
		return nil, err
	}

	pr.LabelAnnotateFrom(ctx, p.Object.ObjectMeta)
	ConfigurePipelineRun(ctx, pr)

	if err := pr.Persist(ctx, cl); err != nil {
		return nil, err
	}

	return pr, nil
}
