package app

import (
	"context"
	"fmt"

	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/relay-core/pkg/expr/evaluate"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureImmutableConfigMapForWorkflowRun(ctx context.Context, cm *ConfigMap, wr *WorkflowRun) error {
	// This implementation manages the underlying object, so no need to retrieve
	// it later.
	lcm := configmap.NewLocalConfigMap(cm.Object)

	params := wr.Object.Spec.Workflow.Parameters.Value()
	for name, value := range wr.Object.Spec.Parameters {
		params[name] = value.Value()
	}

	for name, value := range params {
		if _, err := configmap.NewParameterManager(lcm).Set(ctx, name, value); err != nil {
			return err
		}
	}

	ev := evaluate.NewEvaluator()

	configMapData := make(map[string]string)

	for _, step := range wr.Object.Spec.Workflow.Steps {
		sm := ModelStep(wr, step)

		if len(step.Spec) > 0 {
			r, err := ev.EvaluateAll(ctx, step.Spec.Value())
			if err != nil {
				return errmark.MarkUser(err)
			}

			if _, err := configmap.NewSpecManager(sm, lcm).Set(ctx, r.Value.(map[string]interface{})); err != nil {
				return err
			}
		}

		if env := step.Env.Value(); env != nil {
			em := configmap.NewEnvironmentManager(sm, lcm)

			vars := make(map[string]interface{})
			for name, value := range env {
				r, err := ev.EvaluateAll(ctx, value)
				if err != nil {
					return errmark.MarkUser(err)
				}

				vars[name] = r.Value
			}

			em.Set(ctx, vars)
		}

		if when := step.When.Value(); when != nil {
			r, err := ev.EvaluateAll(ctx, when)
			if err != nil {
				return errmark.MarkUser(err)
			}

			if _, err := configmap.NewConditionManager(sm, lcm).Set(ctx, r.Value); err != nil {
				return err
			}
		}

		if len(step.Input) > 0 {
			configMapData[scriptConfigMapKey(sm)] = model.ScriptForInput(step.Input)
		}
	}

	if len(configMapData) > 0 {
		if _, err := configmap.MutateConfigMap(ctx, lcm, func(cm *corev1.ConfigMap) {
			for name, value := range configMapData {
				cm.Data[name] = value
			}
		}); err != nil {
			return err
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

func configVolumeKey(action model.Action) string {
	return fmt.Sprintf("config-%s-%s", action.Type().Plural, action.Hash())
}

func scriptConfigMapKey(action model.Action) string {
	return fmt.Sprintf("%s.%s.script", action.Type().Plural, action.Hash())
}
