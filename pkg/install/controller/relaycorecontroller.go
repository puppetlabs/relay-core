/*
Copyright 2020 Puppet, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/install/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ownerKey = ".metadata.controller"
)

// RelayCoreReconciler reconciles a RelayCore object
type RelayCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=install.relay.sh,resources=relaycores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.relay.sh,resources=relaycores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;patch

func (r *RelayCoreReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("relaycore", req.NamespacedName)

	relayCore := &installerv1alpha1.RelayCore{}
	if err := r.Get(ctx, req.NamespacedName, relayCore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("reconciling relay core")
	if err := r.desiredOperator(ctx, relayCore); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.desiredMetadataAPI(ctx, relayCore); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RelayCoreReconciler) labels(relayCore *installerv1alpha1.RelayCore) map[string]string {
	return map[string]string{
		"install.relay.sh/relay-core": relayCore.Name,
	}
}

func (r *RelayCoreReconciler) desiredOperator(ctx context.Context, relayCore *installerv1alpha1.RelayCore) error {
	name := fmt.Sprintf("%s-operator", relayCore.Name)
	replicas := int32(1)
	labels := r.labels(relayCore)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: relayCore.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					NodeSelector: relayCore.Spec.Operator.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:            "operator",
							Image:           relayCore.Spec.Operator.Image,
							ImagePullPolicy: relayCore.Spec.Operator.ImagePullPolicy,
							Env:             relayCore.Spec.Operator.Env,
						},
						{
							Name:            "vault",
							Image:           relayCore.Spec.Operator.VaultSidecar.Image,
							ImagePullPolicy: relayCore.Spec.Operator.VaultSidecar.ImagePullPolicy,
							Command: []string{
								"vault",
								"agent",
								"-config=/var/run/vault/config/agent.hcl",
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(relayCore, dep, r.Scheme); err != nil {
		return err
	}

	if err := r.Patch(ctx, dep, client.Apply); err != nil {
		return err
	}

	// srv := &corev1.Service{
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      name,
	// 		Namespace: relayCore.Namespace,
	// 	},
	// 	Spec: &corev1.ServiceSpec{
	// 		Selector: labels,
	// 	},
	// }

	// if err := ctrl.SetControllerReference(&relayCore, srv, r.Scheme); err != nil {
	// 	return err
	// }

	// if err := r.Patch(ctx, srv, client.Apply); err != nil {
	// 	return err
	// }

	return nil
}

func (r *RelayCoreReconciler) desiredMetadataAPI(ctx context.Context, relayCore *installerv1alpha1.RelayCore) error {
	//if err := ctrl.SetControllerReference(&relayCore, &OBJHERE, r.Scheme); err != nil {
	return nil
}

func (r *RelayCoreReconciler) relayCores(obj handler.MapObject) []ctrl.Request {
	ctx := context.Background()
	listOptions := []client.ListOption{
		client.InNamespace(obj.Meta.GetNamespace()),
	}

	var list installerv1alpha1.RelayCoreList
	if err := r.List(ctx, &list, listOptions...); err != nil {
		return nil
	}

	res := make([]ctrl.Request, len(list.Items))

	for i, core := range list.Items {
		res[i].Name = core.GetObjectMeta().GetName()
		res[i].Namespace = core.GetObjectMeta().GetNamespace()
	}

	return res
}

func (r *RelayCoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		&installerv1alpha1.RelayCore{}, ownerKey,
		func(rawObject runtime.Object) []string {
			return []string{}
		}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&installerv1alpha1.RelayCore{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Watches(
			&source.Kind{Type: &installerv1alpha1.RelayCore{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.relayCores),
			}).
		Complete(r)
}
