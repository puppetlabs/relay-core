package run

import (
	"context"
	"fmt"
	"io"

	"github.com/puppetlabs/leg/errmap/pkg/errmap"
	"github.com/puppetlabs/leg/errmap/pkg/errmark"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/errhandler"
	"github.com/puppetlabs/leg/k8sutil/pkg/controller/obj/lifecycle"
	"github.com/puppetlabs/leg/storage"
	relayv1beta1 "github.com/puppetlabs/relay-core/pkg/apis/relay.sh/v1beta1"
	"github.com/puppetlabs/relay-core/pkg/authenticate"
	"github.com/puppetlabs/relay-core/pkg/model"
	"github.com/puppetlabs/relay-core/pkg/obj"
	"github.com/puppetlabs/relay-core/pkg/operator/app"
	"github.com/puppetlabs/relay-core/pkg/operator/dependency"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog/v2"
	"knative.dev/pkg/apis"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const FinalizerName = "workflowrun.finalizers.controller.relay.sh"

type Reconciler struct {
	*dependency.DependencyManager

	Client client.Client
	Scheme *runtime.Scheme

	metrics *controllerObservations
	issuer  authenticate.Issuer
}

func NewReconciler(dm *dependency.DependencyManager) *Reconciler {
	return &Reconciler{
		DependencyManager: dm,

		Client: dm.Manager.GetClient(),
		Scheme: dm.Manager.GetScheme(),

		metrics: newControllerObservations(dm.Metrics),
		issuer: authenticate.IssuerFunc(func(ctx context.Context, claims *authenticate.Claims) (authenticate.Raw, error) {
			raw, err := authenticate.NewKeySignerIssuer(dm.JWTSigner).Issue(ctx, claims)
			if err != nil {
				return nil, err
			}

			return authenticate.NewVaultTransitWrapper(
				dm.VaultClient,
				dm.Config.VaultTransitPath,
				dm.Config.VaultTransitKey,
				authenticate.VaultTransitWrapperWithContext(authenticate.VaultTransitNamespaceContext(claims.KubernetesNamespaceUID)),
			).Wrap(ctx, raw)
		}),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	run := obj.NewRun(req.NamespacedName)
	if ok, err := run.Load(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to load Run")
	} else if !ok {
		// CRD deleted from under us?
		return ctrl.Result{}, nil
	}

	rd := app.NewRunDeps(
		run,
		r.issuer,
		r.Config.MetadataAPIURL,
		app.RunDepsWithEnvironment(r.Config.Environment),
		app.RunDepsWithRuntimeToolsImage(r.Config.RuntimeToolsImage),
		app.RunDepsWithStandaloneMode(r.Config.Standalone),
	)

	loaded, err := rd.Load(ctx, r.Client)
	if err != nil {
		err = errmark.MarkTransientIf(err, errhandler.RuleIsRequired)
		return ctrl.Result{}, errmap.Wrap(err, "failed to load Run dependencies")
	}

	finalized, err := lifecycle.Finalize(ctx, r.Client, FinalizerName, run, func() error {
		_, err := rd.Delete(ctx, r.Client)
		return err
	})
	if err != nil || finalized {
		return ctrl.Result{}, err
	}

	if !loaded.Upstream {
		return ctrl.Result{}, errmark.MarkTransient(fmt.Errorf("waiting on Run upstream dependencies"))
	}

	annotations := rd.Run.Object.GetAnnotations()
	domainID := annotations[model.RelayDomainIDAnnotation]

	var pr *obj.PipelineRun
	err = r.metrics.trackDurationWithOutcome(metricWorkflowRunStartUpDuration, func() error {
		if err := app.ConfigureRunDeps(ctx, rd); err != nil {
			return errmap.Wrap(err, "failed to configure Run dependencies")
		}

		if err := rd.Persist(ctx, r.Client); err != nil {
			return errmap.Wrap(err, "failed to persist Run dependencies")
		}

		if len(rd.Workflow.Object.Spec.Steps) == 0 {
			return nil
		}

		// We only need to build the pipeline when the run is
		// initializing or finalizing. While it's running, constantly persisting
		// pipeline objects just puts strain on the Tekton webhook server.
		//
		// An exception is made of we get out of sync somehow such that the
		// PipelineRun can't load (e.g. someone deletes it from under us).
		switch {
		case run.IsRunning() && !run.IsCancelled():
			pr = obj.NewPipelineRun(
				client.ObjectKey{
					Namespace: rd.WorkflowDeps.TenantDeps.Namespace.Name,
					Name:      run.Key.Name,
				},
			)
			if ok, err := pr.Load(ctx, r.Client); err != nil {
				return errmap.Wrap(err, "failed to load PipelineRun")
			} else if ok {
				break
			}
			fallthrough
		default:
			pipeline, err := app.ApplyPipelineParts(ctx, r.Client, rd)
			if err != nil {
				return errmap.Wrap(err, "failed to apply Pipeline")
			}

			pr, err = app.ApplyPipelineRun(ctx, r.Client, pipeline)
			if err != nil {
				return errmap.Wrap(err, "failed to apply PipelineRun")
			}
		}

		return nil
	}, withAccountIDTrackDurationOption(domainID))
	if err != nil {
		klog.Error(err)
		retryOnError := true
		errmark.IfMarked(err, errmark.User, func(err error) {
			app.ConfigureRunWithSpecificStatus(rd.Run, relayv1beta1.RunSucceeded, corev1.ConditionFalse)

			if ferr := run.PersistStatus(ctx, r.Client); ferr != nil {
				return
			}

			retryOnError = false
		})

		if retryOnError {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	if pr == nil {
		app.ConfigureRunWithSpecificStatus(rd.Run, relayv1beta1.RunSucceeded, corev1.ConditionTrue)

		if err := run.PersistStatus(ctx, r.Client); err != nil {
			return ctrl.Result{}, errmap.Wrap(err, "failed to persist Run status")
		}

		return ctrl.Result{}, nil
	}

	app.ConfigureRun(ctx, rd, pr)

	r.uploadLogs(ctx, run, pr)

	if err := run.PersistStatus(ctx, r.Client); err != nil {
		return ctrl.Result{}, errmap.Wrap(err, "failed to persist Run status")
	}

	return ctrl.Result{}, nil
}

// FIXME Temporary handling for legacy logs
func (r *Reconciler) uploadLogs(ctx context.Context, run *obj.Run, plr *obj.PipelineRun) {
	completed := make(map[string]bool)

	// FIXME Theoretically this can be removed in favor of checking the step status directly
	for _, tr := range plr.Object.Status.TaskRuns {
		if tr.Status == nil || tr.Status.PodName == "" {
			continue
		}

		// TODO: This should support retries, possibly to different log
		// endpoints?
		if cond := tr.Status.GetCondition(apis.ConditionSucceeded); cond == nil || cond.IsUnknown() {
			continue
		}

		completed[tr.Status.PodName] = true
	}

	for i, step := range run.Object.Status.Steps {
		if len(step.Logs) == 0 || step.Logs[0] == nil {
			continue
		}

		if step.Logs[0].Context != "" {
			// Already uploaded.
			continue
		}

		podName := step.Logs[0].Name

		if podName == "" {
			continue
		}

		done, found := completed[podName]
		if !done || !found {
			// Not done yet.
			klog.Infof("Run %s step %q is still progressing, waiting to upload logs", run.Key, step.Name)
			continue
		}

		klog.Infof("Run %s step %q is complete, uploading logs for pod %s", run.Key, step.Name, podName)

		logKey, err := r.uploadLog(ctx, plr.Key.Namespace, podName, "step-step")
		if err != nil {
			klog.Warningf("failed to upload log for Run %s step %q: %+v", run.Key, step.Name, err)
		}

		run.Object.Status.Steps[i].Logs = []*relayv1beta1.Log{
			{
				Name:    podName,
				Context: logKey,
			},
		}
	}
}

func (r *Reconciler) uploadLog(ctx context.Context, namespace, podName, containerName string) (string, error) {
	key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

	// XXX: We can't do this with the dynamic client yet.
	opts := &corev1.PodLogOptions{
		Container: containerName,
	}
	rc, err := r.KubeClient.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer rc.Close()

	storageOpts := storage.PutOptions{
		ContentType: "application/octet-stream",
	}

	err = r.StorageClient.Put(ctx, key, func(w io.Writer) error {
		_, err := io.Copy(w, rc)

		return err
	}, storageOpts)
	if err != nil {
		return "", err
	}

	return key, nil
}
