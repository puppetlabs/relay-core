package app

import (
	"context"
	"fmt"
	"path"

	appsv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/appsv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	rbacv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/rbacv1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/helper"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/k8sutil/pkg/norm"
	"github.com/puppetlabs/relay-core/pkg/apis/install.relay.sh/v1alpha1"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	VaultConfigFileName   = "vault.hcl"
	VaultConfigVolumeName = "vault-config"
	VaultConfigVolumePath = "/var/run/vault/config"
	VaultDataVolumePath   = "/vault/data"
	VaultDataVolumeName   = "data"
	VaultIdentifier       = "vault"
)

type VaultServerBuiltInConfigDeps struct {
	Core               *obj.Core
	ClusterRoleBinding *rbacv1obj.ClusterRoleBinding
	OwnerConfigMap     *corev1obj.ConfigMap
	Service            *corev1obj.Service
	ServiceAccount     *corev1obj.ServiceAccount
	StatefulSet        *appsv1obj.StatefulSet
	Labels             map[string]string
}

func (vd *VaultServerBuiltInConfigDeps) Load(ctx context.Context, cl client.Client) (bool, error) {
	key := client.ObjectKey{
		Name:      VaultIdentifier,
		Namespace: vd.Core.Key.Namespace,
	}

	vd.OwnerConfigMap = corev1obj.NewConfigMap(helper.SuffixObjectKey(key, "owner"))

	vd.ClusterRoleBinding = rbacv1obj.NewClusterRoleBinding(helper.SuffixObjectKey(key, "token-reviewer").Name)
	vd.Service = corev1obj.NewService(key)
	vd.ServiceAccount = corev1obj.NewServiceAccount(key)
	vd.StatefulSet = appsv1obj.NewStatefulSet(key)

	ok, err := lifecycle.Loaders{
		vd.OwnerConfigMap,
		vd.ClusterRoleBinding,
		vd.Service,
		vd.ServiceAccount,
	}.Load(ctx, cl)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (vd *VaultServerBuiltInConfigDeps) Owned(ctx context.Context, owner lifecycle.TypedObject) error {
	return helper.Own(vd.OwnerConfigMap.Object, owner)
}

func (vd *VaultServerBuiltInConfigDeps) Persist(ctx context.Context, cl client.Client) error {
	if err := vd.OwnerConfigMap.Persist(ctx, cl); err != nil {
		return err
	}

	os := []lifecycle.Ownable{
		vd.Service,
		vd.ServiceAccount,
		vd.StatefulSet,
	}

	for _, o := range os {
		if err := vd.OwnerConfigMap.Own(ctx, o); err != nil {
			return err
		}
	}

	objs := []lifecycle.Persister{
		vd.ClusterRoleBinding,
		vd.Service,
		vd.ServiceAccount,
		vd.StatefulSet,
	}

	for _, obj := range objs {
		if err := obj.Persist(ctx, cl); err != nil && !errors.IsAlreadyExists(err) {
			return err
		}
	}

	return nil
}

func (vd *VaultServerBuiltInConfigDeps) Configure(ctx context.Context) error {
	if err := DependencyManager.SetDependencyOf(
		vd.OwnerConfigMap.Object,
		lifecycle.TypedObject{
			Object: vd.Core.Object,
			GVK:    v1alpha1.RelayCoreKind,
		}); err != nil {

		return err
	}

	lafs := []lifecycle.LabelAnnotatableFrom{
		vd.ClusterRoleBinding,
		vd.Service,
		vd.ServiceAccount,
		vd.StatefulSet,
	}

	for _, laf := range lafs {
		for label, value := range vd.Labels {
			lifecycle.Label(ctx, laf, label, value)
		}
	}

	ConfigureClusterRoleBindingWithRoleRef(vd.ServiceAccount, vd.ClusterRoleBinding,
		rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     rbacv1obj.ClusterRoleKind.Kind,
			Name:     "system:auth-delegator",
		},
	)

	ConfigureVaultService(vd, vd.Service)
	ConfigureVaultStatefulSet(vd, vd.StatefulSet)

	return nil
}

func (vd *VaultServerBuiltInConfigDeps) Volumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: VaultConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: vd.Core.Object.Spec.Vault.Server.BuiltIn.ConfigMapRef,
				},
			},
		},
	}
}

func NewVaultServerBuiltInConfigDeps(c *obj.Core) *VaultServerBuiltInConfigDeps {
	return &VaultServerBuiltInConfigDeps{
		Core: c,
		Labels: map[string]string{
			model.RelayInstallerNameLabel: c.Key.Name,
			model.RelayAppNameLabel:       "vault",
			model.RelayAppInstanceLabel:   norm.AnyDNSLabelNameSuffixed("vault-", c.Key.Name),
			model.RelayAppComponentLabel:  "server",
			model.RelayAppManagedByLabel:  "relay-installer",
		},
	}
}

func ConfigureVaultStatefulSet(vd *VaultServerBuiltInConfigDeps, ss *appsv1obj.StatefulSet) {
	template := &ss.Object.Spec.Template.Spec

	if ss.Object.Labels == nil {
		ss.Object.Labels = make(map[string]string)
	}

	for k, v := range vd.Labels {
		ss.Object.Labels[k] = v
	}

	ss.Object.Spec.Template.Labels = vd.Labels
	ss.Object.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: vd.Labels,
	}

	template.RestartPolicy = corev1.RestartPolicyAlways
	template.ServiceAccountName = vd.ServiceAccount.Object.Name

	template.Volumes = vd.Volumes()

	if len(template.Containers) == 0 {
		template.Containers = make([]corev1.Container, 1)
	}

	sc := corev1.Container{}

	ConfigureVaultContainer(vd.Core, &sc)

	template.Containers[0] = sc

	ss.Object.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.OnDeleteStatefulSetStrategyType,
	}

	ss.Object.Spec.ServiceName = vd.Service.Key.Name

	mode := corev1.PersistentVolumeFilesystem
	ss.Object.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: VaultDataVolumeName,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				VolumeMode: &mode,
			},
		},
	}
}

func ConfigureVaultContainer(coreobj *obj.Core, c *corev1.Container) {
	core := coreobj.Object

	c.Name = VaultIdentifier
	c.Image = core.Spec.Vault.Server.BuiltIn.Image
	c.ImagePullPolicy = core.Spec.Vault.Server.BuiltIn.ImagePullPolicy
	c.Resources = core.Spec.Vault.Server.BuiltIn.Resources

	c.Command = []string{"vault", "server",
		fmt.Sprintf("-config=%s", path.Join(VaultConfigVolumePath, VaultConfigFileName))}

	c.SecurityContext = &corev1.SecurityContext{
		Capabilities: &corev1.Capabilities{
			Add: []corev1.Capability{"IPC_LOCK"},
		},
	}

	c.Env = []corev1.EnvVar{
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "status.podIP",
				},
			},
		},
		{
			Name:  "VAULT_API_ADDR",
			Value: "http://$(POD_IP):8200",
		},
		{
			Name:  "VAULT_CLUSTER_ADDR",
			Value: "https://$(POD_IP):8201",
		},
	}

	c.Ports = []corev1.ContainerPort{
		{
			Name:          "vault-client",
			ContainerPort: 8200,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          "vault-cluster",
			ContainerPort: 8201,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	c.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      VaultConfigVolumeName,
			MountPath: VaultConfigVolumePath,
		},
		{
			Name:      VaultDataVolumeName,
			MountPath: VaultDataVolumePath,
		},
	}

	c.LivenessProbe = &corev1.Probe{
		FailureThreshold: 3,
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/v1/sys/health?standbyok=true&uninitcode=200&sealedcode=200",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8200,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
	}

	c.ReadinessProbe = &corev1.Probe{
		FailureThreshold: 3,
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/v1/sys/health?standbyok=true&uninitcode=200&sealedcode=200",
				Port: intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: 8200,
				},
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		SuccessThreshold:    1,
		TimeoutSeconds:      10,
	}
}

func ConfigureVaultService(vd *VaultServerBuiltInConfigDeps, svc *corev1obj.Service) {
	svc.Object.Spec.Selector = vd.Labels

	svc.Object.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "client-port",
			Port:       int32(8200),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8200),
		},
		{
			Name:       "cluster-port",
			Port:       int32(8201),
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.FromInt(8201),
		},
	}
}
