package tenant

import (
	"context"
	"fmt"

	"github.com/puppetlabs/relay-core/pkg/errmark"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/operator/obj"
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

	if tdr.Error != nil {
		return ctrl.Result{}, tdr.Error
	}

	pvc := obj.NewPersistentVolumeClaim(client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tn.Object.Status.Namespace})
	_, err = pvc.Load(ctx, r.Client)
	pvcr := obj.AsPersistentVolumeClaimResult(pvc, err)

	obj.ConfigureTenant(tn, tdr, pvcr)

	if err := tn.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if tn.Ready() {
		if tn.Object.Spec.ToolInjection.VolumeClaimTemplate != nil {
			err := r.cleanupToolInjectionResources(ctx, tn)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	if tn.Object.Spec.ToolInjection.VolumeClaimTemplate != nil {
		pvc, err := r.createReadWriteVolumeClaim(ctx, tn)
		if err != nil {
			return ctrl.Result{}, err
		}

		if pvc.Object.Status.Phase != corev1.ClaimBound {
			return ctrl.Result{Requeue: true}, nil
		}

		job, err := r.initializeVolumeClaim(ctx, tn)
		if err != nil {
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

		pv := obj.NewPersistentVolume(client.ObjectKey{Name: pvc.Object.Spec.VolumeName})
		ok, err := pv.Load(ctx, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !ok ||
			(pv.Object.Spec.GCEPersistentDisk == nil && pv.Object.Spec.HostPath == nil) {
			return ctrl.Result{Requeue: true}, nil
		}

		_, err = r.createReadOnlyVolume(ctx, tn, pv.Object)
		if err != nil {
			return ctrl.Result{}, err
		}

		pvcr := obj.AsPersistentVolumeClaimResult(r.createReadOnlyVolumeClaim(ctx, tn))
		if pvcr.Error != nil {
			return ctrl.Result{}, pvcr.Error
		}

		if pvcr.PersistentVolumeClaim.Object.Status.Phase != corev1.ClaimBound {
			return ctrl.Result{Requeue: true}, nil
		}

		obj.ConfigureTenant(tn, tdr, pvcr)

		if err := tn.PersistStatus(ctx, r.Client); err != nil {
			return ctrl.Result{}, err
		}

		if tn.Ready() {
			err := r.cleanupToolInjectionResources(ctx, tn)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if !tn.Ready() {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) cleanupToolInjectionResources(ctx context.Context, tn *obj.Tenant) error {
	job := obj.NewJob(client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce, Namespace: tn.Object.Status.Namespace})
	_, err := job.Load(ctx, r.Client)
	if err != nil {
		return err
	}

	_, err = job.Delete(ctx, r.Client, client.PropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createReadOnlyVolume(ctx context.Context, tn *obj.Tenant, pv *corev1.PersistentVolume) (*obj.PersistentVolume, error) {
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

	key := client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany}
	pvno, err := obj.ApplyPersistentVolume(ctx, r.Client, key, pvn)
	if err != nil {
		return nil, err
	}

	return pvno, nil
}

func (r *Reconciler) createReadOnlyVolumeClaim(ctx context.Context, tn *obj.Tenant) (*obj.PersistentVolumeClaim, error) {
	pvcn := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany,
			Namespace:   tn.Object.Status.Namespace,
			Annotations: tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Annotations,
			Labels:      tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany},
			Resources:        tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.Resources,
			StorageClassName: tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.StorageClassName,
		},
	}

	key := client.ObjectKey{Name: pvcn.GetName(), Namespace: pvcn.GetNamespace()}
	pvcno, err := obj.ApplyPersistentVolumeClaim(ctx, r.Client, key, pvcn)
	if err != nil {
		return nil, err
	}

	return pvcno, err
}

func (r *Reconciler) createReadWriteVolumeClaim(ctx context.Context, tn *obj.Tenant) (*obj.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce,
			Namespace: tn.Object.Status.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.Resources,
			StorageClassName: tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.StorageClassName,
		},
	}

	key := client.ObjectKey{Name: pvc.GetName(), Namespace: pvc.GetNamespace()}
	pvco, err := obj.ApplyPersistentVolumeClaim(ctx, r.Client, key, pvc)
	if err != nil {
		return nil, err
	}

	return pvco, err
}

func (r *Reconciler) initializeVolumeClaim(ctx context.Context, tn *obj.Tenant) (*obj.Job, error) {
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
		ImagePullPolicy: corev1.PullAlways,
	}

	defaultLimit := int32(1)
	root := int64(0)

	j := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce,
			Namespace: tn.Object.Status.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: model.ToolInjectionMountName,
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
									ClaimName: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce,
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

	key := client.ObjectKey{Name: j.GetName(), Namespace: j.GetNamespace()}
	job, err := obj.ApplyJob(ctx, r.Client, key, j)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return job, nil
}
