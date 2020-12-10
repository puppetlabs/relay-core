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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	ownerKey                = ".metadata.controller"
	jwtSigningKeyDirPath    = "/var/run/secrets/puppet/relay/jwt"
	jwtSigningKeyPath       = "/var/run/secrets/puppet/relay/jwt/private-key.pem"
	webhookTLSDirPath       = "/var/run/secrets/puppet/relay/webhook-tls"
	vaultAgentConfigDirPath = "/var/run/vault/config"
	vaultAgentSATokenPath   = "/var/run/secrets/kubernetes.io/serviceaccount@vault"
	metadataAPITLSDirPath   = "/var/run/secrets/puppet/relay/tls"
)

// RelayCoreReconciler reconciles a RelayCore object
type RelayCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=install.relay.sh,resources=relaycores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.relay.sh,resources=relaycores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core,resources=configmaps;limitranges;serviceaccounts;services;secrets;namespaces;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch;patch;create;update;delete
// +kubebuilder:rbac:groups=core,resources=pods;pods/log,verbs=get;list;watch
// +kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns;taskruns;pipelines;tasks;conditions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch;extensions,resources=jobs,verbs=get;list;watch;patch;create;update;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfiguration,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nebula.puppet.com,resources=workflowruns;workflowruns/status,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups=relay.sh,resources=tenants;tenants/status;webhooktriggers;webhooktriggers/status,verbs=get;list;watch;patch;update
// +kubebuilder:rbac:groups=serving.knative.dev,resources=services,verbs=get;list;watch;create;update;patch;delete

func (r *RelayCoreReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("relaycore", req.NamespacedName)

	relayCore := &installerv1alpha1.RelayCore{}
	if err := r.Get(ctx, req.NamespacedName, relayCore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Validate and set defaults

	// here we ensure all the required *SecretName secrets exist. If the
	// required secrets don't exist, we must requeue and check the next cycle.
	if !relayCore.Spec.Operator.GenerateJWTSigningKey {
		if relayCore.Spec.Operator.JWTSigningKeySecretName != nil {
			key := client.ObjectKey{
				Name:      *relayCore.Spec.Operator.JWTSigningKeySecretName,
				Namespace: relayCore.Namespace,
			}

			if err := r.checkSecret(ctx, key); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if relayCore.Spec.SentryDSNSecretName != nil {
		key := client.ObjectKey{
			Name:      *relayCore.Spec.SentryDSNSecretName,
			Namespace: relayCore.Namespace,
		}

		if err := r.checkSecret(ctx, key); err != nil {
			return ctrl.Result{}, err
		}
	}

	if relayCore.Spec.MetadataAPI.TLSSecretName != nil {
		key := client.ObjectKey{
			Name:      *relayCore.Spec.MetadataAPI.TLSSecretName,
			Namespace: relayCore.Namespace,
		}

		if err := r.checkSecret(ctx, key); err != nil {
			return ctrl.Result{}, err
		}
	}

	log.Info("reconciling relay core")

	// reconcile objects for each manager type

	log.Info("reconciling relay-log-service")
	lssm := newLogServiceStateManager(relayCore, r, log)
	if err := lssm.reconcile(ctx); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling relay-operator")
	osm := newOperatorStateManager(relayCore, r, log)
	if err := osm.reconcile(ctx); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("reconciling relay-metadata-api")
	msm := newMetadataAPIStateManager(relayCore, r, log)
	if err := msm.reconcile(ctx); err != nil {
		return ctrl.Result{}, err
	}

	// TODO hook up the controller-runtime event system to change status to
	// running when everything is working
	log.Info("relay core created; updating status")
	relayCore.Status.Status = installerv1alpha1.StatusCreated
	if err := r.updateStatus(ctx, relayCore); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RelayCoreReconciler) updateStatus(ctx context.Context, relayCore *installerv1alpha1.RelayCore) error {
	return r.Status().Update(ctx, relayCore)
}

func (r *RelayCoreReconciler) checkSecret(ctx context.Context, key types.NamespacedName) error {
	if err := r.Get(ctx, key, &corev1.Secret{}); err != nil {
		return err
	}

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
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.ConfigMap{}).
		Watches(
			&source.Kind{Type: &installerv1alpha1.RelayCore{}},
			&handler.EnqueueRequestsFromMapFunc{
				ToRequests: handler.ToRequestsFunc(r.relayCores),
			}).
		Complete(r)
}
