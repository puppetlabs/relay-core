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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	installerv1alpha1 "github.com/puppetlabs/relay-core/pkg/install/api/v1alpha1"
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
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;watch;patch;create;update
func (r *RelayCoreReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("relaycore", req.NamespacedName)

	relayCore := &installerv1alpha1.RelayCore{}
	if err := r.Get(ctx, req.NamespacedName, relayCore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

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

	osm := newOperatorStateManager(relayCore, r.labels(relayCore))
	operatorName := fmt.Sprintf("%s-operator", relayCore.Name)

	operatorVaultConfigMap := corev1.ConfigMap{}
	operatorVaultConfigMap.Name = fmt.Sprintf("%s-vault", operatorName)
	operatorVaultConfigMap.Namespace = relayCore.Namespace

	_, err := ctrl.CreateOrUpdate(ctx, r, &operatorVaultConfigMap, func() error {
		osm.vaultAgentManager.configMap(&operatorVaultConfigMap)

		return ctrl.SetControllerReference(relayCore, &operatorVaultConfigMap, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	operatorVaultServiceAccount := corev1.ServiceAccount{}
	operatorVaultServiceAccount.Name = fmt.Sprintf("%s-vault", operatorName)
	operatorVaultServiceAccount.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &operatorVaultServiceAccount, func() error {
		return ctrl.SetControllerReference(relayCore, &operatorVaultServiceAccount, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	ovk, err := client.ObjectKeyFromObject(&operatorVaultServiceAccount)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, ovk, &operatorVaultServiceAccount); err != nil {
		return ctrl.Result{}, err
	}

	operatorServiceAccount := corev1.ServiceAccount{}
	operatorServiceAccount.Name = operatorName
	operatorServiceAccount.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &operatorServiceAccount, func() error {
		return ctrl.SetControllerReference(relayCore, &operatorServiceAccount, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	operatorDeployment := appsv1.Deployment{}
	operatorDeployment.Name = operatorName
	operatorDeployment.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &operatorDeployment, func() error {
		vaultToken := operatorVaultServiceAccount.Secrets[0]
		osm.deployment(&operatorDeployment, vaultToken.Name)

		return ctrl.SetControllerReference(relayCore, &operatorDeployment, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	msm := newMetadataAPIStateManager(relayCore, r.labels(relayCore))
	metadataAPIName := fmt.Sprintf("%s-metadata-api", relayCore.Name)

	metadataAPIVaultConfigMap := corev1.ConfigMap{}
	metadataAPIVaultConfigMap.Name = fmt.Sprintf("%s-vault", metadataAPIName)
	metadataAPIVaultConfigMap.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &metadataAPIVaultConfigMap, func() error {
		msm.vaultAgentManager.configMap(&metadataAPIVaultConfigMap)

		return ctrl.SetControllerReference(relayCore, &metadataAPIVaultConfigMap, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	metadataAPIVaultServiceAccount := corev1.ServiceAccount{}
	metadataAPIVaultServiceAccount.Name = fmt.Sprintf("%s-vault", metadataAPIName)
	metadataAPIVaultServiceAccount.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &metadataAPIVaultServiceAccount, func() error {
		return ctrl.SetControllerReference(relayCore, &metadataAPIVaultServiceAccount, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	mvk, err := client.ObjectKeyFromObject(&metadataAPIVaultServiceAccount)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, mvk, &metadataAPIVaultServiceAccount); err != nil {
		return ctrl.Result{}, err
	}

	metadataAPIServiceAccount := corev1.ServiceAccount{}
	metadataAPIServiceAccount.Name = metadataAPIName
	metadataAPIServiceAccount.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &metadataAPIServiceAccount, func() error {
		return ctrl.SetControllerReference(relayCore, &metadataAPIServiceAccount, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	metadataAPIDeployment := appsv1.Deployment{}
	metadataAPIDeployment.Name = metadataAPIName
	metadataAPIDeployment.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &metadataAPIDeployment, func() error {
		vaultToken := metadataAPIVaultServiceAccount.Secrets[0]
		msm.deployment(&metadataAPIDeployment, vaultToken.Name)

		return ctrl.SetControllerReference(relayCore, &metadataAPIDeployment, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	metadataAPIService := corev1.Service{}
	metadataAPIService.Name = metadataAPIName
	metadataAPIService.Namespace = relayCore.Namespace

	_, err = ctrl.CreateOrUpdate(ctx, r, &metadataAPIService, func() error {
		msm.httpService(&metadataAPIService)

		return ctrl.SetControllerReference(relayCore, &metadataAPIService, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RelayCoreReconciler) updateStatus(ctx context.Context, relayCore *installerv1alpha1.RelayCore) error {
	return r.Status().Update(ctx, relayCore)
}

func (r *RelayCoreReconciler) labels(relayCore *installerv1alpha1.RelayCore) map[string]string {
	return map[string]string{
		"install.relay.sh/relay-core":  relayCore.Name,
		"app.kubernetes.io/name":       "relay-operator",
		"app.kubernetes.io/managed-by": "relay-install-operator",
	}
}

func (r *RelayCoreReconciler) desiredOperatorSigningKeySecret(ctx context.Context, relaycore *installerv1alpha1.RelayCore) error {
	// if the secret exists for this deployment, then we don't need to create it again
	// otherwise we generate a keypair and create a new secret

	return nil
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
