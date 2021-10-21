package app

import (
	"context"
	"fmt"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	corev1 "k8s.io/api/core/v1"
)

func ConfigureImmutableConfigMapForWebhookTrigger(ctx context.Context, cm *corev1obj.ConfigMap, wt *obj.WebhookTrigger) error {
	tm := ModelWebhookTrigger(wt)

	// This implementation manages the underlying object, so no need to retrieve
	// it later.
	lcm := configmap.NewLocalConfigMap(cm.Object)

	if len(wt.Object.Spec.Spec) > 0 {
		if _, err := configmap.NewSpecManager(tm, lcm).Set(ctx, wt.Object.Spec.Spec.Value()); err != nil {
			return err
		}
	}

	if env := wt.Object.Spec.Env.Value(); env != nil {
		em := configmap.NewEnvironmentManager(ModelWebhookTrigger(wt), lcm)

		vars := make(map[string]interface{})
		for name, value := range env {
			vars[name] = value
		}

		if _, err := em.Set(ctx, vars); err != nil {
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

func ConfigureImmutableConfigMapForWorkflowRun(ctx context.Context, cm *corev1obj.ConfigMap, rd *RunDeps) error {
	// This implementation manages the underlying object, so no need to retrieve
	// it later.
	lcm := configmap.NewLocalConfigMap(cm.Object)

	params := make(map[string]*relayv1beta1.Unstructured)

	wp := rd.Workflow.Object.Spec.Parameters
	for _, value := range wp {
		if value != nil {
			params[value.Name] = nil
			if value.Value != nil {
				params[value.Name] = value.Value.DeepCopy()
			}
		}
	}

	wrp := rd.WorkflowRun.Object.Spec.Parameters
	for name, value := range wrp {
		params[name] = value.DeepCopy()
	}

	for name, value := range params {
		if _, err := configmap.NewParameterManager(lcm).Set(ctx, name, value); err != nil {
			return err
		}
	}

	configMapData := make(map[string]string)

	for _, step := range rd.Workflow.Object.Spec.Steps {
		sm := ModelStep(rd.WorkflowRun, step)

		if len(step.Spec) > 0 {
			if _, err := configmap.NewSpecManager(sm, lcm).Set(ctx, step.Spec.Value()); err != nil {
				return err
			}
		}

		if env := step.Env.Value(); env != nil {
			em := configmap.NewEnvironmentManager(sm, lcm)

			vars := make(map[string]interface{})
			for name, value := range env {
				vars[name] = value
			}

			if _, err := em.Set(ctx, vars); err != nil {
				return err
			}
		}

		if step.When != nil {
			if when := step.When.Value(); when != nil {
				if _, err := configmap.NewConditionManager(sm, lcm).Set(ctx, when); err != nil {
					return err
				}
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

func ConfigureMutableConfigMapForWorkflowRun(ctx context.Context, cm *corev1obj.ConfigMap, wr *obj.WorkflowRun) error {
	lcm := configmap.NewLocalConfigMap(cm.Object)

	for stepName, state := range wr.Object.Spec.State.Steps {
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
