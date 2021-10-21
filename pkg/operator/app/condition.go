package app

import (
	"context"

	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/obj"
	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func ConfigureCondition(ctx context.Context, c *obj.Condition, rd *RunDeps, ws *relayv1beta1.Step) error {
	if err := rd.AnnotateStepToken(ctx, &c.Object.ObjectMeta, ws); err != nil {
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
						Value: rd.MetadataAPIURL.String(),
					},
				},
			},
			Script: ConditionScript,
		},
	}

	return nil
}

type ConditionSet struct {
	Deps *RunDeps
	List []*obj.Condition
	idx  map[string]int
}

var _ lifecycle.LabelAnnotatableFrom = &ConditionSet{}
var _ lifecycle.Loader = &ConditionSet{}
var _ lifecycle.Ownable = &ConditionSet{}
var _ lifecycle.Persister = &ConditionSet{}

func (cs *ConditionSet) LabelAnnotateFrom(ctx context.Context, from metav1.Object) {
	for _, c := range cs.List {
		c.LabelAnnotateFrom(ctx, from)
	}
}

func (cs *ConditionSet) Load(ctx context.Context, cl client.Client) (bool, error) {
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

func (cs *ConditionSet) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	for _, cond := range cs.List {
		if err := cond.Owned(ctx, owner); err != nil {
			return err
		}
	}

	return nil
}

func (cs *ConditionSet) Persist(ctx context.Context, cl client.Client) error {
	for _, cond := range cs.List {
		if err := cond.Persist(ctx, cl); err != nil {
			return err
		}
	}

	return nil
}

func (cs *ConditionSet) GetByStepName(stepName string) (*obj.Condition, bool) {
	idx, found := cs.idx[stepName]
	if !found {
		return nil, false
	}

	return cs.List[idx], true
}

func NewConditionSet(rd *RunDeps) *ConditionSet {
	cs := &ConditionSet{
		Deps: rd,
		idx:  make(map[string]int),
	}

	var i int
	for _, ws := range rd.Workflow.Object.Spec.Steps {
		if ws.When == nil || ws.When.Value() == nil {
			continue
		}

		cs.List = append(cs.List, obj.NewCondition(
			ModelStepObjectKey(
				client.ObjectKey{
					Namespace: rd.WorkflowDeps.TenantDeps.Namespace.Name,
					Name:      rd.Run.Key.Name,
				},
				ModelStep(rd.Run, ws),
			),
		))
		cs.idx[ws.Name] = i
		i++
	}

	return cs
}

func ConfigureConditionSet(ctx context.Context, cs *ConditionSet) error {
	for _, ws := range cs.Deps.Workflow.Object.Spec.Steps {
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
