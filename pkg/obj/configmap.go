package obj

import (
	"context"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/expr/resolve"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	ConfigMapKind = corev1.SchemeGroupVersion.WithKind("ConfigMap")
)

type ConfigMap struct {
	Key    client.ObjectKey
	Object *corev1.ConfigMap
}

var _ Persister = &ConfigMap{}
var _ Loader = &ConfigMap{}
var _ Deleter = &ConfigMap{}
var _ Ownable = &ConfigMap{}
var _ LabelAnnotatableFrom = &ConfigMap{}

func (cm *ConfigMap) Persist(ctx context.Context, cl client.Client) error {
	return CreateOrUpdate(ctx, cl, cm.Key, cm.Object)
}

func (cm *ConfigMap) Load(ctx context.Context, cl client.Client) (bool, error) {
	return GetIgnoreNotFound(ctx, cl, cm.Key, cm.Object)
}

func (cm *ConfigMap) Delete(ctx context.Context, cl client.Client) (bool, error) {
	return DeleteIgnoreNotFound(ctx, cl, cm.Object)
}

func (cm *ConfigMap) Owned(ctx context.Context, owner Owner) error {
	return Own(cm.Object, owner)
}

func (cm *ConfigMap) Own(ctx context.Context, other Ownable) error {
	return other.Owned(ctx, Owner{GVK: ConfigMapKind, Object: cm.Object})
}

func (cm *ConfigMap) LabelAnnotateFrom(ctx context.Context, from metav1.ObjectMeta) {
	CopyLabelsAndAnnotations(&cm.Object.ObjectMeta, from)
}

func NewConfigMap(key client.ObjectKey) *ConfigMap {
	return &ConfigMap{
		Key:    key,
		Object: &corev1.ConfigMap{},
	}
}

func ConfigureImmutableConfigMapForWebhookTrigger(ctx context.Context, cm *ConfigMap, wt *WebhookTrigger) error {
	tm := ModelWebhookTrigger(wt)

	// This implementation manages the underlying object, so no need to retrieve
	// it later.
	lcm := configmap.NewLocalConfigMap(cm.Object)

	ev := evaluate.NewEvaluator()

	if len(wt.Object.Spec.Spec) > 0 {
		r, err := ev.EvaluateAll(ctx, wt.Object.Spec.Spec.Value())
		if err != nil {
			return err
		}

		if _, err := configmap.NewSpecManager(tm, lcm).Set(ctx, r.Value.(map[string]interface{})); err != nil {
			return err
		}
	}

	if len(wt.Object.Spec.Input) > 0 {
		if _, err := configmap.MutateConfigMap(ctx, lcm, func(cm *corev1.ConfigMap) {
			cm.Data[scriptConfigMapKey(tm)] = model.ScriptForInput(wt.Object.Spec.Input)
		}); err != nil {
			return err
		}
	}

	return nil
}

func ConfigureImmutableConfigMapForWorkflowRun(ctx context.Context, cm *ConfigMap, wr *WorkflowRun) error {
	// This implementation manages the underlying object, so no need to retrieve
	// it later.
	lcm := configmap.NewLocalConfigMap(cm.Object)

	specParamers := wr.Object.Spec.Workflow.Parameters.Value()
	runParameters := wr.Object.Spec.Parameters.Value()

	ev := evaluate.NewEvaluator(
		evaluate.WithParameterTypeResolver(resolve.ParameterTypeResolverFunc(func(ctx context.Context, name string) (interface{}, error) {
			if p, ok := runParameters[name]; ok {
				return p, nil
			} else if p, ok := specParamers[name]; ok {
				return p, nil
			}

			return nil, &resolve.ParameterNotFoundError{Name: name}
		})),
	)

	for _, step := range wr.Object.Spec.Workflow.Steps {
		if len(step.Spec) > 0 {
			r, err := ev.EvaluateAll(ctx, step.Spec.Value())
			if err != nil {
				return err
			}

			if _, err := configmap.NewSpecManager(ModelStep(wr, step), lcm).Set(ctx, r.Value.(map[string]interface{})); err != nil {
				return err
			}
		}

		if when := step.When.Value(); when != nil {
			r, err := ev.EvaluateAll(ctx, when)
			if err != nil {
				return err
			}

			if _, err := configmap.NewConditionManager(ModelStep(wr, step), lcm).Set(ctx, r.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

func ConfigureMutableConfigMapForWorkflowRun(ctx context.Context, cm *ConfigMap, wr *WorkflowRun) error {
	lcm := configmap.NewLocalConfigMap(cm.Object)

	for stepName, state := range wr.Object.State.Steps {
		sm := configmap.NewStateManager(ModelStepFromName(wr, stepName), lcm)

		for name, value := range state {
			if _, err := sm.Set(ctx, name, value.Value()); err != nil {
				return err
			}
		}
	}

	return nil
}

func scriptConfigMapKey(action model.Action) string {
	return fmt.Sprintf("%s.%s.script", action.Type().Plural, action.Hash())
}
