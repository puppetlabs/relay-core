package tenant

import (
	"context"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	batchv1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/batchv1"
	corev1obj "github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/api/corev1"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/operator/config"
	"github.com/puppetlabs/relay-core/pkg/util/image"
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

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	tn := obj.NewTenant(req.NamespacedName)
	if ok, err := tn.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load dependencies")
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	deps := app.NewTenantDeps(tn, app.TenantDepsWithStandaloneMode(r.Config.Standalone))
	if _, err := deps.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load dependencies")
	}

	finalized, err := lifecycle.Finalize(ctx, r.Client, FinalizerName, tn, func() error {
		_, err := deps.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	if _, err := deps.DeleteStale(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to delete stale dependencies")
	}

	app.ConfigureTenantDeps(ctx, deps)

	tdr := app.AsTenantDepsResult(deps, deps.Persist(ctx, r.Client))

	if tdr.Error != nil {
		return ctrl.Result{}, tdr.Error
	}

	pvcROX := corev1obj.NewPersistentVolumeClaim(client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tn.Object.Status.Namespace})
	_, err = pvcROX.Load(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, err
	}

	app.ConfigureTenant(tn, tdr, app.AsPersistentVolumeClaimResult(pvcROX, err))

	if err := tn.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, err
	}

	if tn.Ready() {
		return ctrl.Result{}, nil
	}

	if tn.Object.Spec.ToolInjection.VolumeClaimTemplate != nil &&
		pvcROX.Object.Status.Phase != corev1.ClaimBound {
		annotations := tn.Object.GetAnnotations()
		if annotations == nil {
			annotations = map[string]string{}
		}

		volume, ok := annotations[model.RelayControllerToolsVolumeAnnotation]
		if !ok {
			reference, ok := annotations[model.RelayControllerToolInjectionImageDigestAnnotation]
			if !ok {
				tii := model.DefaultToolInjectionImage
				if r.Config.ToolInjectionImage != "" {
					tii = r.Config.ToolInjectionImage
				}

				reference, err = image.ValidateImage(tii)
				if err != nil {
					return ctrl.Result{}, err
				}

				original := tn.Copy()
				lifecycle.Annotate(ctx, tn, model.RelayControllerToolInjectionImageDigestAnnotation, reference)
				err = obj.NewTenantPatcher(tn, original).Persist(ctx, r.Client)
				if err != nil {
					return ctrl.Result{}, err
				}
			}

			pvcRWO, err := r.createReadWriteVolumeClaim(ctx, tn)
			if err != nil {
				return ctrl.Result{}, err
			}

			if pvcRWO.Object.Status.Phase != corev1.ClaimBound {
				return ctrl.Result{Requeue: true}, nil
			}

			pv := corev1obj.NewPersistentVolume(pvcRWO.Object.Spec.VolumeName)
			ok, err = pv.Load(ctx, r.Client)
			if err != nil {
				return ctrl.Result{}, err
			}

			if !ok || pv.Object.Status.Phase != corev1.VolumeBound {
				return ctrl.Result{Requeue: true}, nil
			}

			job, err := r.initializeVolumeClaim(ctx, reference, tn)
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

			volume = pv.Object.GetName()
			original := tn.Copy()
			lifecycle.Annotate(ctx, tn, model.RelayControllerToolsVolumeAnnotation, volume)
			err = obj.NewTenantPatcher(tn, original).Persist(ctx, r.Client)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		err := r.cleanupToolInjectionResources(ctx, tn)
		if err != nil {
			return ctrl.Result{}, err
		}

		pv := corev1obj.NewPersistentVolume(volume)
		ok, err = pv.Load(ctx, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !ok {
			return ctrl.Result{Requeue: true}, nil
		}

		if pv.Object.Status.Phase != corev1.VolumeBound {
			original := pv.Copy()
			lifecycle.Label(ctx, pv, model.RelayControllerToolInjectionVolumeLabel, tn.Object.GetName())
			pv.Object.Spec.ClaimRef = nil
			pv.Object.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadOnlyMany}

			err = corev1obj.NewPersistentVolumePatcher(pv, original).Persist(ctx, r.Client)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		pvcROX := corev1obj.NewPersistentVolumeClaim(client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadOnlyMany, Namespace: tn.Object.Status.Namespace})
		ok, err = pvcROX.Load(ctx, r.Client)
		if err != nil {
			return ctrl.Result{}, err
		}

		if !ok {
			pvList := &corev1.PersistentVolumeList{}
			err = r.Client.List(ctx, pvList, client.MatchingFields{
				"status.phase": string(corev1.VolumeAvailable),
			})
			if err != nil {
				return ctrl.Result{}, err
			}

			if len(pvList.Items) <= 0 {
				return ctrl.Result{Requeue: true}, nil
			}

			pvcROX, err = r.createReadOnlyVolumeClaim(ctx, tn)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		_, err = pvcROX.Load(ctx, r.Client)
		app.ConfigureTenant(tn, tdr, app.AsPersistentVolumeClaimResult(pvcROX, err))

		if err := tn.PersistStatus(ctx, r.Client); err != nil {
			return ctrl.Result{}, err
		}
	}

	if !tn.Ready() {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

func (r *Reconciler) cleanupToolInjectionResources(ctx context.Context, tn *obj.Tenant) error {
	job := batchv1obj.NewJob(client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce, Namespace: tn.Object.GetNamespace()})
	_, err := job.Load(ctx, r.Client)
	if err != nil {
		return err
	}

	_, err = job.Delete(ctx, r.Client, lifecycle.DeleteWithPropagationPolicy(metav1.DeletePropagationForeground))
	if err != nil {
		return err
	}

	pvcRWO := corev1obj.NewPersistentVolumeClaim(client.ObjectKey{Name: tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce, Namespace: tn.Object.GetNamespace()})
	_, err = pvcRWO.Load(ctx, r.Client)
	if err != nil {
		return err
	}

	_, err = pvcRWO.Delete(ctx, r.Client)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createReadOnlyVolumeClaim(ctx context.Context, tn *obj.Tenant) (*corev1obj.PersistentVolumeClaim, error) {
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
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					model.RelayControllerToolInjectionVolumeLabel: tn.Object.GetName(),
				},
			},
		},
	}

	key := client.ObjectKey{Name: pvcn.GetName(), Namespace: pvcn.GetNamespace()}
	pvcno, err := app.ApplyPersistentVolumeClaim(ctx, r.Client, key, pvcn)
	if err != nil {
		return nil, err
	}

	return pvcno, err
}

func (r *Reconciler) createReadWriteVolumeClaim(ctx context.Context, tn *obj.Tenant) (*corev1obj.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tn.Object.GetName() + model.ToolInjectionVolumeClaimSuffixReadWriteOnce,
			Namespace: tn.Object.GetNamespace(),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:        tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.Resources,
			StorageClassName: tn.Object.Spec.ToolInjection.VolumeClaimTemplate.Spec.StorageClassName,
		},
	}

	key := client.ObjectKey{Name: pvc.GetName(), Namespace: pvc.GetNamespace()}
	pvco, err := app.ApplyPersistentVolumeClaim(ctx, r.Client, key, pvc)
	if err != nil {
		return nil, err
	}

	return pvco, err
}

func (r *Reconciler) initializeVolumeClaim(ctx context.Context, image string, tn *obj.Tenant) (*batchv1obj.Job, error) {
	container := corev1.Container{
		Name:    model.ToolInjectionMountName,
		Image:   image,
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
			Namespace: tn.Object.GetNamespace(),
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
	job, err := app.ApplyJob(ctx, r.Client, key, j)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return job, nil
}
