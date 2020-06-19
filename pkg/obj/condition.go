package obj

import (
	"context"

	nebulav1 "github.com/puppetlabs/relay-core/pkg/apis/nebula.puppet.com/v1"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConditionImage  = "relaysh/core:latest"
	ConditionScript = `#!/bin/bash
JQ="${JQ:-jq}"

CONDITIONS_URL="${CONDITIONS_URL:-conditions}"
VALUE_NAME="${VALUE_NAME:-success}"
POLLING_INTERVAL="${POLLING_INTERVAL:-5s}"
POLLING_ITERATIONS="${POLLING_ITERATIONS:-1080}"

for i in $(seq ${POLLING_ITERATIONS}); do
	CONDITIONS=$(curl "$METADATA_API_URL/${CONDITIONS_URL}")
	VALUE=$(echo $CONDITIONS | $JQ --arg value "$VALUE_NAME" -r '.[$value]')
	if [ -n "${VALUE}" ]; then
	if [ "$VALUE" = "true" ]; then
		exit 0
	fi
	if [ "$VALUE" = "false" ]; then
		exit 1
	fi
	fi
	sleep ${POLLING_INTERVAL}
done

exit 1
`
)

type Condition struct {
	Key    client.ObjectKey
	Object *tektonv1alpha1.Condition
}

var _ Persister = &Condition{}
var _ Loader = &Condition{}
var _ Ownable = &Condition{}

func (c *Condition) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, c.Key, c.Object)
}

func (c *Condition) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, c.Key, c.Object)
}

func (c *Condition) Owned(ctx context.Context, owner Owner) error {
	return Own(c.Object, owner)
}

func NewCondition(key client.ObjectKey) *Condition {
	return &Condition{
		Key:    key,
		Object: &tektonv1alpha1.Condition{},
	}
}

func ConfigureCondition(ctx context.Context, c *Condition, wrd *WorkflowRunDeps, ws *nebulav1.WorkflowStep) error {
	if err := wrd.AnnotateStepToken(ctx, &c.Object.ObjectMeta, ws); err != nil {
		return err
	}

	c.Object.Spec = tektonv1alpha1.ConditionSpec{
		Check: tektonv1beta1.Step{
			Container: corev1.Container{
				Image: ConditionImage,
				Name:  "condition",
				Env: []corev1.EnvVar{
					{
						Name:  "METADATA_API_URL",
						Value: wrd.MetadataAPIURL.String(),
					},
				},
			},
			Script: ConditionScript,
		},
	}

	return nil
}

type Conditions struct {
	Deps *WorkflowRunDeps
	List []*Condition
	idx  map[string]int
}

var _ Persister = &Conditions{}
var _ Loader = &Conditions{}
var _ Ownable = &Conditions{}

func (cs *Conditions) Persist(ctx context.Context, cl client.Client) error {
	for _, cond := range cs.List {
		if err := cond.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (cs *Conditions) Load(ctx context.Context, cl client.Client) (bool, error) {
	all := true

	for _, cond := range cs.List {
		ok, err := cond.Load(ctx, cl)
		if err != nil {
			return false, err
		} else if !ok {
			all = false
		}
	}

	return all, nil
}

func (cs *Conditions) Owned(ctx context.Context, owner Owner) error {
	for _, cond := range cs.List {
		if err := cond.Owned(ctx, owner); err != nil {
			return err
		}
	}

	return nil
}

func (cs *Conditions) GetByStepName(stepName string) (*Condition, bool) {
	idx, found := cs.idx[stepName]
	if !found {
		return nil, false
	}

	return cs.List[idx], true
}

func NewConditions(wrd *WorkflowRunDeps) *Conditions {
	cs := &Conditions{
		Deps: wrd,
		idx:  make(map[string]int),
	}

	var i int
	for _, ws := range wrd.WorkflowRun.Object.Spec.Workflow.Steps {
		if ws.When.Value() == nil {
			continue
		}

		cs.List = append(cs.List, NewCondition(ModelStepObjectKey(wrd.WorkflowRun.Key, ModelStep(wrd.WorkflowRun, ws))))
		cs.idx[ws.Name] = i
		i++
	}

	return cs
}

func ConfigureConditions(ctx context.Context, cs *Conditions) error {
	if err := cs.Deps.WorkflowRun.Own(ctx, cs); err != nil {
		return err
	}

	for _, ws := range cs.Deps.WorkflowRun.Object.Spec.Workflow.Steps {
		cond, found := cs.GetByStepName(ws.Name)
		if !found {
			continue
		}

		if err := ConfigureCondition(ctx, cond, cs.Deps, ws); err != nil {
			return err
		}
	}

	return nil
}
