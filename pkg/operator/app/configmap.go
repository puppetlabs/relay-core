package app

import (
	"context"
	"fmt"

	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/relspec/pkg/evaluate"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/manager/configmap"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/spec"
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

func ConfigureImmutableConfigMapForRun(ctx context.Context, cm *corev1obj.ConfigMap, rd *RunDeps) error {
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

	wrp := rd.Run.Object.Spec.Parameters
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
		sm := ModelStep(rd.Run, step)

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

		when := enrichWhenConditions(ctx, step)
		if len(when) > 0 {
			if _, err := configmap.NewConditionManager(sm, lcm).Set(ctx, when); err != nil {
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

func ConfigureMutableConfigMapForRun(ctx context.Context, cm *corev1obj.ConfigMap, r *obj.Run) error {
	lcm := configmap.NewLocalConfigMap(cm.Object)

	for stepName, state := range r.Object.Spec.State.Steps {
		sm := configmap.NewStateManager(ModelStepFromName(r, stepName), lcm)

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

func enrichWhenConditions(ctx context.Context, step *relayv1beta1.Step) []interface{} {
	when := make([]interface{}, 0)

	useDefaultDependencyFlow := map[string]bool{}
	for _, dependency := range step.DependsOn {
		useDefaultDependencyFlow[dependency] = true
	}

	if step.When != nil && step.When.Value() != nil {
		existing, ok := step.When.Value().([]interface{})
		if ok {
			when = append(when, existing...)
		} else {
			when = append(when, step.When.Value())
		}

		if r, err := evaluate.EvaluateAll(ctx, spec.NewEvaluator(), step.When.Value()); err == nil {
			if r != nil && r.References != nil && r.References.Statuses != nil {
				for _, v := range r.References.Statuses.UsedReferences() {
					useDefaultDependencyFlow[v.ID().Action] = false
				}
			}
		}
	}

	for dependency, useDefault := range useDefaultDependencyFlow {
		if useDefault {
			// TODO Implement a more programmatic way of adding expressions without using the explicit expression language.
			// TODO Consider adding internal conditions to separate generated expressions from user-defined ones.
			when = append(when,
				fmt.Sprintf("${steps.'%s'.%s}",
					dependency, model.StatusPropertySucceeded.String()))
		}
	}

	return when
}
