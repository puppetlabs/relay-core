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
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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
	ownerKey          = ".metadata.controller"
	jwtSigningKeyPath = "/var/run/secrets/puppet/relay/jwt/key.pem"
)

// RelayCoreReconciler reconciles a RelayCore object
type RelayCoreReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=install.relay.sh,resources=relaycores,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=install.relay.sh,resources=relaycores/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch;create;update
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;patch;create;update

func (r *RelayCoreReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("relaycore", req.NamespacedName)

	relayCore := &installerv1alpha1.RelayCore{}
	if err := r.Get(ctx, req.NamespacedName, relayCore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info("reconciling relay core")
	if err := r.desiredOperator(ctx, req, relayCore); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.desiredMetadataAPI(ctx, req, relayCore); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RelayCoreReconciler) labels(relayCore *installerv1alpha1.RelayCore) map[string]string {
	return map[string]string{
		"install.relay.sh/relay-core":  relayCore.Name,
		"app.kubernetes.io/name":       "relay-operator",
		"app.kubernetes.io/managed-by": "relay-install-operator",
	}
}

func (r *RelayCoreReconciler) desiredOperator(ctx context.Context, req ctrl.Request, relayCore *installerv1alpha1.RelayCore) error {
	name := fmt.Sprintf("%s-operator", relayCore.Name)
	labels := r.labels(relayCore)
	labels["app.kubernetes.io/name"] = "operator"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: relayCore.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
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
							Env:             r.desiredOperatorEnv(relayCore),
							Command:         r.desiredOperatorCommand(relayCore),
						},
						r.vaultSidecarContainer(relayCore),
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(relayCore, dep, r.Scheme); err != nil {
		return err
	}

	result, err := ctrl.CreateOrUpdate(ctx, r, dep, func() error {
		return nil
	})

	if err != nil {
		return err
	}

	r.Log.Info("reconciled operator deployment", "result", result)

	return nil
}

func (r *RelayCoreReconciler) desiredOperatorCommand(relayCore *installerv1alpha1.RelayCore) []string {
	cmd := []string{"relay-operator"}

	cmd = append(cmd, "-storage-addr", relayCore.Spec.Operator.StorageAddr)

	if relayCore.Spec.Operator.Standalone {
		cmd = append(cmd, "-standalone")
	}

	if relayCore.Spec.Operator.MetricsEnabled {
		cmd = append(cmd, "-metrics-enabled", "-metrics-server-bind-addr", "0.0.0.0:3050")
	}

	if relayCore.Spec.Operator.TenantSandboxingRuntimeClassName != nil {
		cmd = append(cmd,
			"-tenant-sandboxing",
			"-tenant-sandbox-runtime-class-name",
			relayCore.Spec.Operator.TenantSandboxingRuntimeClassName,
		)
	}

	metadataAPIURL := fmt.Sprintf("http://%s-metadata-api.%s.svc.cluster.local", relayCore.GetName(), relayCore.GetNamespace())
	if relayCore.Spec.MetadataAPI.URL != nil {
		metadataAPIURL = *relayCore.Spec.MetadataAPI.URL
	}

	// TODO make these configurable
	cmd = append(cmd,
		"-environment",
		relayCore.Spec.Environment,
		"-num-workers",
		strconv.Itoa(int(relayCore.Spec.Operator.Workers)),
		// TODO convert to generateJWTSigningKey field
		"-jwt-signing-key-file",
		jwtSigningKeyPath,
		"-vault-transit-path",
		relayCore.Spec.Vault.TransitPath,
		"-vault-transit-key",
		relayCore.Spec.Vault.TransitKey,
		"-metadata-api-url",
		metadataAPIURL,
		"-webhook-server-key-dir",
		"/var/run/secrets/puppet/relay/webhook-tls",
		"-sentry-dsn",
		"$(RELAY_OPERATOR_SENTRY_DSN)",
		"-dynamic-rbac-binding",
		"-tool-injection-image",
		relayCore.Spec.Operator.ToolInjection.Image,
	)

	return cmd
}

func (r *RelayCoreReconciler) desiredOperatorEnv(relayCore *installerv1alpha1.RelayCore) []corev1.EnvVar {
	env := []corev1.EnvVar{{Name: "VAULT_ADDR", Value: "http://localhost:8200"}}

	if relayCore.Spec.Operator.Env != nil {
		env = append(env, relayCore.Spec.Operator.Env...)
	}

	return env
}

func (r *RelayCoreReconciler) desiredMetadataAPI(ctx context.Context, req ctrl.Request, relayCore *installerv1alpha1.RelayCore) error {
	name := fmt.Sprintf("%s-metadata-api", relayCore.Name)
	labels := r.labels(relayCore)
	labels["app.kubernetes.io/name"] = "metadata-api"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: relayCore.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &relayCore.Spec.MetadataAPI.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					NodeSelector: relayCore.Spec.MetadataAPI.NodeSelector,
					Containers: []corev1.Container{
						{
							Name:            "metadata-api",
							Image:           relayCore.Spec.MetadataAPI.Image,
							ImagePullPolicy: relayCore.Spec.MetadataAPI.ImagePullPolicy,
							Env:             r.desiredMetadataAPIEnv(relayCore),
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: int32(7000),
									Protocol:      corev1.ProtocolTCP,
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromString("https"),
										Scheme: corev1.URISchemeHTTPS,
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromString("https"),
										Scheme: corev1.URISchemeHTTPS,
									},
								},
							},
						},
						r.vaultSidecarContainer(relayCore),
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(relayCore, dep, r.Scheme); err != nil {
		return err
	}

	result, err := ctrl.CreateOrUpdate(ctx, r, dep, func() error {
		return nil
	})

	if err != nil {
		return err
	}

	r.Log.Info("reconciled metadata-api deployment", "result", result)

	srv := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: relayCore.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "https",
					Port:       int32(443),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString("https"),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(relayCore, srv, r.Scheme); err != nil {
		return err
	}

	result, err = ctrl.CreateOrUpdate(ctx, r, srv, func() error {
		return nil
	})

	r.Log.Info("reconciled metadata-api service", "result", result)

	return nil
}

func (r *RelayCoreReconciler) desiredMetadataAPIEnv(relayCore *installerv1alpha1.RelayCore) []corev1.EnvVar {
	env := []corev1.EnvVar{{Name: "VAULT_ADDR", Value: "http://localhost:8200"}}

	if relayCore.Spec.MetadataAPI.Env != nil {
		env = append(env, relayCore.Spec.MetadataAPI.Env...)
	}

	return env
}

func (r *RelayCoreReconciler) vaultSidecarContainer(relayCore *installerv1alpha1.RelayCore) corev1.Container {
	c := corev1.Container{
		Name:            "vault",
		Image:           relayCore.Spec.Vault.Sidecar.Image,
		ImagePullPolicy: relayCore.Spec.Vault.Sidecar.ImagePullPolicy,
		Command: []string{
			"vault",
			"agent",
			"-config=/var/run/vault/config/agent.hcl",
		},
		Resources: relayCore.Spec.Vault.Sidecar.Resources,
	}

	// volumeMounts := []corev1.VolumeMount{
	// 	relayCore.Spec.VaultSidecar.ConfigVolumeMounts,
	// 	relayCore.Spec.VaultSidecar.ServiceAccountVolumeMount,
	// }

	// c.VolumeMounts = volumeMounts

	return c
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
