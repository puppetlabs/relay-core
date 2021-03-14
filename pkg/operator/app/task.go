package app

import (
	"context"
	"path"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/admission"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ConfigureTask(ctx context.Context, t *obj.Task, wrd *WorkflowRunDeps, ws *nebulav1.WorkflowStep) error {
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
			// We can't use RunAsUser et al. here because they don't allow write
			// access to the container filesystem. Eventually, we'll use gVisor
			// to protect us here.
			AllowPrivilegeEscalation: func(b bool) *bool { return &b }(false),
		},
	}

	command := ws.Command
	args := ws.Args

	if len(ws.Input) > 0 {
		sm := ModelStep(wrd.WorkflowRun, ws)

		found := false
		config := configVolumeKey(sm)
		for _, volume := range t.Object.Spec.Volumes {
			if volume.Name == config {
				found = true
				break
			}
		}

		if !found {
			t.Object.Spec.Volumes = append(t.Object.Spec.Volumes, corev1.Volume{
				Name: config,
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
			})
		}

		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      config,
			ReadOnly:  true,
			MountPath: "/var/run/puppet/relay/config",
		})

		command = "/var/run/puppet/relay/config/input-script"
		args = []string{}
	}

	// TODO Reference the tool injection from the tenant (once this is available)
	// For now, we'll assume an explicit tenant reference implies the use of the entrypoint handling
	if tr := wrd.WorkflowRun.Object.Spec.TenantRef; tr != nil && wrd.ToolInjectionPool != nil {
		ep, err := entrypoint.ImageEntrypoint(image, []string{command}, args)
		if err != nil {
			return err
		}

		container.Command = []string{path.Join(model.ToolsMountPath, ep.Entrypoint)}
		container.Args = ep.Args

		pvcName := checkoutObjectKey(client.ObjectKey{
			Namespace: wrd.Namespace.Name,
			Name:      tr.Name,
		}, wrd.ToolInjectionPool.Key)
		helper.Annotate(&t.Object.ObjectMeta, admission.ToolsVolumeClaimAnnotation, pvcName.Name)
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
		List: make([]*obj.Task, len(wrd.WorkflowRun.Object.Spec.Workflow.Steps)),
	}

	for i, ws := range wrd.WorkflowRun.Object.Spec.Workflow.Steps {
		ts.List[i] = obj.NewTask(ModelStepObjectKey(wrd.WorkflowRun.Key, ModelStep(wrd.WorkflowRun, ws)))
	}

	return ts
}

func ConfigureTaskSet(ctx context.Context, ts *TaskSet) error {
	if err := ts.Deps.WorkflowRun.Own(ctx, ts); err != nil {
		return err
	}

	for i, ws := range ts.Deps.WorkflowRun.Object.Spec.Workflow.Steps {
		if err := ConfigureTask(ctx, ts.List[i], ts.Deps, ws); err != nil {
			return err
		}
	}

	return nil
}
