package tenant

import (
	"context"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/config"
	"github.com/puppetlabs/relay-core/pkg/errmark"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "tenant.finalizers.controller.relay.sh"

type Reconciler struct {
	Client client.Client
	Config *config.WorkflowControllerConfig
}

func NewReconciler(client client.Client, cfg *config.WorkflowControllerConfig) *Reconciler {
	return &Reconciler{
		Client: client,
		Config: cfg,
	}
}

func (r *Reconciler) Reconcile(req ctrl.Request) (result ctrl.Result, err error) {
	ctx := context.Background()

	tn := obj.NewTenant(req.NamespacedName)
	if ok, err := tn.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmark.MapLast(err, func(err error) error {
			return fmt.Errorf("failed to load dependencies: %+v", err)
		})
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	deps := obj.NewTenantDeps(tn)
	if _, err := deps.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmark.MapLast(err, func(err error) error {
			return fmt.Errorf("failed to load dependencies: %+v", err)
		})
	}

	finalized, err := obj.Finalize(ctx, r.Client, FinalizerName, tn, func() error {
		_, err := deps.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	if _, err := deps.DeleteStale(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmark.MapLast(err, func(err error) error {
			return fmt.Errorf("failed to delete stale dependencies: %+v", err)
		})
	}

	obj.ConfigureTenantDeps(ctx, deps)

	tdr := obj.AsTenantDepsResult(deps, deps.Persist(ctx, r.Client))

	obj.ConfigureTenant(tn, tdr, []batchv1.JobCondition{})

	if err := tn.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if tn.Object.Spec.ToolInjection.VolumeClaimTemplate == nil {
		if !tn.Ready() {
			return ctrl.Result{Requeue: true}, nil
		}

		return ctrl.Result{}, nil
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      deps.Tenant.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce,
			Namespace: tn.Object.Spec.NamespaceTemplate.Metadata.GetName(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.Resources,
			StorageClassName: tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.StorageClassName,
		},
	}

	key := client.ObjectKey{Name: deps.Tenant.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce, Namespace: deps.Tenant.Object.Spec.NamespaceTemplate.Metadata.GetName()}
	pvco, err := obj.ApplyPersistentVolumeClaim(ctx, r.Client, key, pvc)
	if err != nil {
		return ctrl.Result{}, err
	}

	if pvco.Object.Spec.VolumeName == "" || pvco.Object.Status.Phase != corev1.ClaimBound {
		return ctrl.Result{Requeue: true}, nil
	}

	pv := &corev1.PersistentVolume{}

	if err := r.Client.Get(ctx, client.ObjectKey{Name: pvco.Object.Spec.VolumeName}, pv); k8serrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if pv.Spec.GCEPersistentDisk == nil && pv.Spec.HostPath == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	pvn := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany,
		},
		Spec: corev1.PersistentVolumeSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
			Capacity:         pv.Spec.Capacity,
			StorageClassName: pv.Spec.StorageClassName,
		},
	}

	pvn.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}

	if pv.Spec.GCEPersistentDisk != nil {
		pvn.Spec.PersistentVolumeSource = corev1.PersistentVolumeSource{
			GCEPersistentDisk: pv.Spec.GCEPersistentDisk,
		}
	} else if pv.Spec.HostPath != nil {
		pvn.Spec.PersistentVolumeSource = corev1.PersistentVolumeSource{
			HostPath: pv.Spec.HostPath,
		}
	}

	key = client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany}
	_, err = obj.ApplyPersistentVolume(ctx, r.Client, key, pvn)
	if err != nil {
		return ctrl.Result{}, err
	}

	pvcn := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvn.GetName(),
			Namespace: tn.Object.Spec.NamespaceTemplate.Metadata.GetName(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
			Resources:        pvco.Object.Spec.Resources,
			StorageClassName: pvco.Object.Spec.StorageClassName,
		},
	}

	key = client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tn.Object.Spec.NamespaceTemplate.Metadata.GetName()}
	_, err = obj.ApplyPersistentVolumeClaim(ctx, r.Client, key, pvcn)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Get(ctx, client.ObjectKey{Namespace: pvcn.GetNamespace(), Name: pvcn.GetName()}, pvcn); k8serrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if pvcn.Spec.VolumeName == "" {
		return ctrl.Result{Requeue: true}, nil
	}

	tii := model.DefaultToolInjectionImage
	if r.Config.ToolInjectionImage != "" {
		tii = r.Config.ToolInjectionImage
	}

	container := corev1.Container{
		Name:    model.ToolInjectionMountName,
		Image:   tii,
		Command: []string{"cp"},
		Args:    []string{"-r", model.ToolInjectionImagePath, model.ToolInjectionMountPath},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      model.ToolInjectionMountName,
				MountPath: model.ToolInjectionMountPath,
			},
		},
	}

	defaultLimit := int32(1)
	root := int64(0)

	j := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany,
			Namespace: tn.Object.Spec.NamespaceTemplate.Metadata.GetName(),
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      model.ToolInjectionMountName,
					Namespace: tn.Object.Spec.NamespaceTemplate.Metadata.GetName(),
				},
				Spec: corev1.PodSpec{
					Containers:    []corev1.Container{container},
					RestartPolicy: corev1.RestartPolicyNever,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: &root,
					},
					Volumes: []corev1.Volume{
						{
							Name: model.ToolInjectionMountName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvco.Object.GetName(),
								},
							},
						},
					},
				},
			},
			BackoffLimit: &defaultLimit,
			Completions:  &defaultLimit,
			Parallelism:  &defaultLimit,
		},
	}

	key = client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tn.Object.Spec.NamespaceTemplate.Metadata.GetName()}
	job, err := obj.ApplyJob(ctx, r.Client, key, j)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return ctrl.Result{}, err
	}

	complete := false
	failed := false
	for _, cond := range job.Object.Status.Conditions {
		switch cond.Type {
		case batchv1.JobComplete:
			switch cond.Status {
			case corev1.ConditionTrue:
				complete = true
			}
		case batchv1.JobFailed:
			switch cond.Status {
			case corev1.ConditionTrue:
				failed = true
			}
		}
	}

	if !complete && !failed {
		return ctrl.Result{Requeue: true}, nil
	}

	obj.ConfigureTenant(tn, tdr, job.Object.Status.Conditions)

	if err := tn.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if !tn.Ready() {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}
