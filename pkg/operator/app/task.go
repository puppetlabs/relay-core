package app

import (
	"context"
	"path"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigureTask(ctx context.Context, t *obj.Task, wrd *WorkflowRunDeps, ws *relayv1beta1.Step) error {
	image := ws.Image
	if image == "" {
		image = model.DefaultImage
	}

	container := corev1.Container{
		Name:            "step",
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Env: []corev1.EnvVar{
			{
				Name:  "CI",
				Value: "true",
			},
			{
				Name:  "RELAY",
				Value: "true",
			},
			{
				Name:  "METADATA_API_URL",
				Value: wrd.MetadataAPIURL.String(),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: func(b bool) *bool { return &b }(false),
		},
	}

	if wrd.LimitRange != nil {
		container.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceEphemeralStorage: resource.MustParse("20Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceEphemeralStorage: resource.MustParse("20Gi"),
			},
		}
	}

	command := ws.Command
	args := ws.Args

	if len(ws.Input) > 0 {
		sm := ModelStep(wrd.WorkflowRun, ws)

		vol := corev1.Volume{
			Name: configVolumeKey(sm),
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: wrd.ImmutableConfigMap.Key.Name,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  scriptConfigMapKey(sm),
							Path: "input-script",
							Mode: func(i int32) *int32 { return &i }(0755),
						},
					},
				},
			},
		}

		t.SetVolume(vol)
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      vol.Name,
			ReadOnly:  true,
			MountPath: "/var/run/puppet/relay/config",
		})

		command = "/var/run/puppet/relay/config/input-script"
		args = []string{}
	}

	if wrd.ToolInjectionCheckout.Satisfied() {
		ep, err := entrypoint.ImageEntrypoint(image, []string{command}, args)
		if err != nil {
			return err
		}

		t.SetWorkspace(tektonv1beta1.WorkspaceDeclaration{
			Name:      ToolsWorkspaceName,
			MountPath: model.ToolsMountPath,
			ReadOnly:  true,
		})
		container.Command = []string{path.Join(model.ToolsMountPath, ep.Entrypoint)}
		container.Args = ep.Args
	} else {
		if command != "" {
			container.Command = []string{command}
		}

		if len(args) > 0 {
			container.Args = args
		}
	}

	if err := wrd.AnnotateStepToken(ctx, &t.Object.ObjectMeta, ws); err != nil {
		return err
	}

	t.Object.Spec.Steps = []tektonv1beta1.Step{
		{
			Container: container,
		},
	}

	return nil
}

type TaskSet struct {
	Deps *WorkflowRunDeps
	List []*obj.Task
}

var _ lifecycle.LabelAnnotatableFrom = &TaskSet{}
var _ lifecycle.Loader = &TaskSet{}
var _ lifecycle.Ownable = &TaskSet{}
var _ lifecycle.Persister = &TaskSet{}

func (ts *TaskSet) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	for _, t := range ts.List {
		t.LabelAnnotateFrom(ctx, from)
	}
}

func (ts *TaskSet) Load(ctx context.Context, cl client.Client) (bool, error) {
	all := true

	for _, t := range ts.List {
		ok, err := t.Load(ctx, cl)
		if err != nil {
			return false, err
		} else if !ok {
			all = false
		}
	}

	return all, nil
}

func (ts *TaskSet) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	for _, t := range ts.List {
		if err := t.Owned(ctx, owner); err != nil {
			return err
		}
	}

	return nil
}

func (ts *TaskSet) Persist(ctx context.Context, cl client.Client) error {
	for _, t := range ts.List {
		if err := t.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func NewTaskSet(wrd *WorkflowRunDeps) *TaskSet {
	ts := &TaskSet{
		Deps: wrd,
		List: make([]*obj.Task, len(wrd.Workflow.Object.Spec.Steps)),
	}

	for i, ws := range wrd.Workflow.Object.Spec.Steps {
		ts.List[i] = obj.NewTask(ModelStepObjectKey(wrd.WorkflowRun.Key, ModelStep(wrd.WorkflowRun, ws)))
	}

	return ts
}

func ConfigureTaskSet(ctx context.Context, ts *TaskSet) error {
	if err := ts.Deps.WorkflowRun.Own(ctx, ts); err != nil {
		return err
	}

	for i, ws := range ts.Deps.Workflow.Object.Spec.Steps {
		if err := ConfigureTask(ctx, ts.List[i], ts.Deps, ws); err != nil {
			return err
		}
	}

	return nil
}
