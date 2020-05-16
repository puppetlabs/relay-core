package obj

import (
	"context"
	"strings"

	nebulav1 "github.com/puppetlabs/nebula-tasks/pkg/apis/nebula.puppet.com/v1"
	"github.com/puppetlabs/nebula-tasks/pkg/model"
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

	step := tektonv1beta1.Step{
		Container: corev1.Container{
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
		},
	}

	if len(ws.Input) > 0 {
		script := strings.Join(ws.Input, "\n")
		if !strings.HasPrefix(script, model.Shebang) {
			script = model.DefaultInterpreter + "\n" + script
		}

		step.Script = script
	} else {
		if len(ws.Command) > 0 {
			step.Container.Command = []string{ws.Command}
		}

		if len(ws.Args) > 0 {
			step.Container.Args = ws.Args
		}
	}

	if err := wrd.AnnotateStepToken(ctx, &t.Object.ObjectMeta, ws); err != nil {
		return err
	}

	t.Object.Spec.Steps = []tektonv1beta1.Step{step}

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
