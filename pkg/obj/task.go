package obj

import (
	"context"
	"path"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/relay-core/pkg/entrypoint"
	"github.com/puppetlabs/relay-core/pkg/model"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Task struct {
	Key    client.ObjectKey
	Object *tektonv1beta1.Task
}

var _ Persister = &Task{}
var _ Loader = &Task{}
var _ Ownable = &Task{}

func (t *Task) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, t.Key, t.Object)
}

func (t *Task) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, t.Key, t.Object)
}

func (t *Task) Owned(ctx context.Context, owner Owner) error {
	return Own(t.Object, owner)
}

func NewTask(key client.ObjectKey) *Task {
	return &Task{
		Key:    key,
		Object: &tektonv1beta1.Task{},
	}
}

func ConfigureTask(ctx context.Context, t *Task, wrd *WorkflowRunDeps, ws *nebulav1.WorkflowStep) error {
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
	if wrd.WorkflowRun.Object.Spec.TenantRef != nil {
		ep, err := entrypoint.ImageEntrypoint(image, []string{command}, args)
		if err != nil {
			return err
		}

		container.Command = []string{path.Join(model.ToolInjectionMountPath, ep.Entrypoint)}
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

	// TODO Reference the tool injection from the tenant (once this is available)
	// For now, we'll assume an explicit tenant reference implies the use of the tool injection suite
	if wrd.WorkflowRun.Object.Spec.TenantRef != nil {
		claim := wrd.WorkflowRun.Object.Spec.TenantRef.Name + model.ToolInjectionVolumeClaimSuffixReadOnlyMany
		Annotate(&t.Object.ObjectMeta, model.RelayControllerToolsVolumeClaimAnnotation, claim)
	}

	t.Object.Spec.Steps = []tektonv1beta1.Step{
		{
			Container: container,
		},
	}

	return nil
}

type Tasks struct {
	Deps *WorkflowRunDeps
	List []*Task
}

var _ Persister = &Tasks{}
var _ Loader = &Tasks{}
var _ Ownable = &Tasks{}

func (ts *Tasks) Persist(ctx context.Context, cl client.Client) error {
	for _, t := range ts.List {
		if err := t.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (ts *Tasks) Load(ctx context.Context, cl client.Client) (bool, error) {
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

func (ts *Tasks) Owned(ctx context.Context, owner Owner) error {
	for _, t := range ts.List {
		if err := t.Owned(ctx, owner); err != nil {
			return err
		}
	}

	return nil
}

func NewTasks(wrd *WorkflowRunDeps) *Tasks {
	ts := &Tasks{
		Deps: wrd,
		List: make([]*Task, len(wrd.WorkflowRun.Object.Spec.Workflow.Steps)),
	}

	for i, ws := range wrd.WorkflowRun.Object.Spec.Workflow.Steps {
		ts.List[i] = NewTask(ModelStepObjectKey(wrd.WorkflowRun.Key, ModelStep(wrd.WorkflowRun, ws)))
	}

	return ts
}

func ConfigureTasks(ctx context.Context, ts *Tasks) error {
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
